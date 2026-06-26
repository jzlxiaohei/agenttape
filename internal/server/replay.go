package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"tracelab/internal/event"
	"tracelab/internal/normalize"
	"tracelab/internal/source"
	"tracelab/internal/source/httpcap"
	"tracelab/internal/store"
)

// replayResp is the result of re-sending a request to upstream.
type replayResp struct {
	Status         int                           `json:"status"`
	DurationMS     int64                         `json:"duration_ms"`
	Normalized     *normalize.NormalizedEnvelope `json:"normalized,omitempty"`
	NormalizeError string                        `json:"normalize_error,omitempty"`
	// ResponseBody is the raw upstream body (capped), so the UI can show what went
	// wrong on a non-2xx — error JSON normalizes to empty final_text, which would
	// otherwise render as a blank "—". Honest over polished; matches the raw-bytes
	// ethos. Truncated marks that maxReplayBody clipped it.
	ResponseBody string `json:"response_body,omitempty"`
	Truncated    bool   `json:"truncated,omitempty"`
}

// maxReplayBody caps the raw body returned inline so a huge (or streamed) response
// can't bloat the JSON. 64 KiB is plenty to read an error payload.
const maxReplayBody = 64 << 10

var replayClient = &http.Client{Timeout: 10 * time.Minute}

// capBody returns the body as a string, clipped to maxReplayBody, and whether it
// was clipped.
func capBody(b []byte) (string, bool) {
	if len(b) > maxReplayBody {
		return string(b[:maxReplayBody]), true
	}
	return string(b), false
}

// handleReplay re-sends a captured completion to its upstream — optionally with a
// modified request body — and returns the freshly normalized result. Not
// persisted. Auth comes from the session's in-memory headers, so only sessions
// captured in THIS process can be replayed (credentials are never on disk).
func (s *Server) handleReplay(st *store.Store, w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	detail, err := st.GetEvent(id)
	if err == store.ErrNoRows {
		http.Error(w, "event not found", http.StatusNotFound)
		return
	}
	if err != nil {
		httpError(w, err)
		return
	}
	if !detail.IsCompletion {
		http.Error(w, "only completion requests can be replayed", http.StatusBadRequest)
		return
	}
	headers := s.Sessions.AuthFor(detail.SessionID)
	if headers == nil {
		http.Error(w, replayCredentialConflictMessage(s, detail.SessionID), http.StatusConflict)
		return
	}

	var in struct {
		Body *string `json:"body"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)
	var reqBody []byte
	if in.Body != nil {
		reqBody = []byte(*in.Body)
	} else {
		path, err := st.RawFilePath(id, "request_body")
		if err != nil {
			httpError(w, err)
			return
		}
		if reqBody, err = os.ReadFile(path); err != nil {
			httpError(w, err)
			return
		}
	}

	out, err := s.executeReplay(r.Context(), detail.Method, detail.Target, reqBody, headers, detail.SessionID)
	if err != nil {
		http.Error(w, "replay failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, out)
}

// executeReplay sends one request to upstream with the given auth headers and
// normalizes the response via the same registry capture uses. Shared by
// event-replay and replay-lib case runs.
func (s *Server) executeReplay(ctx context.Context, method, target string, body []byte, auth http.Header, sessionID string) (*replayResp, error) {
	upReq, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	applyReplayHeaders(upReq.Header, auth)
	upReq.Header.Set("Accept-Encoding", "identity")

	started := time.Now().UTC()
	resp, err := replayClient.Do(upReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	completed := time.Now().UTC()

	ev := buildReplayEvent(sessionID, target, method, auth.Get("Content-Type"), body, resp, respBody, started, completed)
	out := &replayResp{Status: resp.StatusCode, DurationMS: completed.Sub(started).Milliseconds()}
	out.ResponseBody, out.Truncated = capBody(respBody)
	if env, err := s.reg.Normalize(ev); err != nil {
		out.NormalizeError = err.Error()
	} else {
		out.Normalized = env
	}
	return out, nil
}

// executeCaseThroughSession sends a replay-library case through the same launch
// session proxy path real clients use: /s/<token>/<endpoint>. The selected
// session supplies both upstream routing and credentials, while the case supplies
// only the request body plus session-relative endpoint.
func (s *Server) executeCaseThroughSession(ctx context.Context, proxyBase string, sess *httpcap.Session, method, endpoint string, body []byte, auth http.Header) (*replayResp, error) {
	proxyTarget := strings.TrimRight(proxyBase, "/") + "/s/" + sess.Token + "/" + strings.TrimPrefix(endpoint, "/")
	upstreamTarget := strings.TrimRight(sess.Upstream, "/") + "/" + strings.TrimPrefix(endpoint, "/")

	upReq, err := http.NewRequestWithContext(ctx, method, proxyTarget, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	applyReplayHeaders(upReq.Header, auth)
	upReq.Header.Set("Accept-Encoding", "identity")

	started := time.Now().UTC()
	resp, err := replayClient.Do(upReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	completed := time.Now().UTC()

	ev := buildReplayEvent(sess.ID, upstreamTarget, method, auth.Get("Content-Type"), body, resp, respBody, started, completed)
	out := &replayResp{Status: resp.StatusCode, DurationMS: completed.Sub(started).Milliseconds()}
	out.ResponseBody, out.Truncated = capBody(respBody)
	if env, err := s.reg.Normalize(ev); err != nil {
		out.NormalizeError = err.Error()
	} else {
		out.Normalized = env
	}
	return out, nil
}

// applyReplayHeaders copies auth/content headers onto the replay request, dropping
// ones the client must own per-send.
func applyReplayHeaders(dst, src http.Header) {
	for k, vs := range src {
		switch strings.ToLower(k) {
		case "content-length", "host", "accept-encoding", "connection", "transfer-encoding":
			continue
		}
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

// buildReplayEvent shapes the upstream response into the same SourceEvent the
// capture proxy produces, so the normalize registry handles it unchanged.
func buildReplayEvent(sessionID, target, method, reqContentType string, reqBody []byte, resp *http.Response, respBody []byte, started, completed time.Time) *event.SourceEvent {
	id := source.RandomID()
	ev := event.New(id, event.KindHTTPExchange, event.SourceRef{
		Kind:    event.SourceCapture,
		Adapter: "replay",
		Mode:    "replay",
	})
	ev.Correlation = event.Correlation{SessionID: sessionID}
	ev.Timing = event.Timing{
		StartedAt:   started.Format(time.RFC3339Nano),
		CompletedAt: completed.Format(time.RFC3339Nano),
		DurationMS:  completed.Sub(started).Milliseconds(),
	}
	reqArt := event.NewRawArtifact(id+":req", event.RoleRequestBody, reqBody)
	reqArt.MediaType = reqContentType
	respArt := event.NewRawArtifact(id+":resp", event.RoleResponseBody, respBody)
	respArt.MediaType = resp.Header.Get("Content-Type")
	ev.RawArtifacts = []event.RawArtifact{reqArt, respArt}
	ev.Capture = &event.CaptureEvent{
		Protocol: "http",
		Method:   method,
		URL:      target,
		Target:   target,
		Request: event.HTTPMessage{
			ContentType:    reqContentType,
			BodyArtifactID: reqArt.ID,
			BodySizeBytes:  int64(len(reqBody)),
		},
		Response: event.HTTPMessage{
			StatusCode:     resp.StatusCode,
			ContentType:    resp.Header.Get("Content-Type"),
			BodyArtifactID: respArt.ID,
			BodySizeBytes:  int64(len(respBody)),
		},
	}
	return &ev
}
