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
	"tracelab/internal/store"
)

// replayResp is the result of re-sending a request to upstream.
type replayResp struct {
	Status         int                           `json:"status"`
	DurationMS     int64                         `json:"duration_ms"`
	Normalized     *normalize.NormalizedEnvelope `json:"normalized,omitempty"`
	NormalizeError string                        `json:"normalize_error,omitempty"`
}

var replayClient = &http.Client{Timeout: 10 * time.Minute}

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
		http.Error(w, "no in-memory credentials for this session (captured before a restart or in another process); launch a new session to replay", http.StatusConflict)
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
