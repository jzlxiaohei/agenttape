package httpcap

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jzlxiaohei/agenttape/internal/event"
	"github.com/jzlxiaohei/agenttape/internal/source"
)

// Proxy is the reverse-proxy capture adapter. Requests arrive as
// /s/<token>/<upstream-path> and are forwarded to the session's upstream while
// the exchange is recorded and emitted.
type Proxy struct {
	sessions *Sessions
	emit     source.Emitter
	client   *http.Client
}

// NewProxy builds a capture proxy.
func NewProxy(sessions *Sessions, emit source.Emitter) *Proxy {
	return &Proxy{
		sessions: sessions,
		emit:     emit,
		client:   &http.Client{Timeout: 10 * time.Minute},
	}
}

// Name identifies the adapter.
func (p *Proxy) Name() string { return "httpcap" }

// Handler returns the proxy as an http.Handler (mounted at /s/).
func (p *Proxy) Handler() http.Handler { return http.StripPrefix("/s/", p) }

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token, rest := splitToken(r.URL.Path)
	sess := p.sessions.Lookup(token)
	if sess == nil {
		http.Error(w, "agenttape: unknown session token", http.StatusBadGateway)
		return
	}

	upURL := sess.Upstream + "/" + rest
	if r.URL.RawQuery != "" {
		upURL += "?" + r.URL.RawQuery
	}

	started := time.Now().UTC()
	reqBody, _ := io.ReadAll(r.Body)
	_ = r.Body.Close()

	upReq, err := http.NewRequestWithContext(r.Context(), r.Method, upURL, bytes.NewReader(reqBody))
	if err != nil {
		http.Error(w, "agenttape: build upstream request: "+err.Error(), http.StatusBadGateway)
		return
	}
	copyHeaders(upReq.Header, r.Header)
	// API-key launch mode: the agent sent a placeholder key; swap in the real auth
	// held only in memory for this session, so the key never reaches the agent.
	if inject := p.sessions.InjectFor(sess.ID); inject != nil {
		for k, vs := range inject {
			upReq.Header.Del(k)
			for _, v := range vs {
				upReq.Header.Add(k, v)
			}
		}
	}
	// Force identity so upstream returns uncompressed bodies — no decompression
	// dependency, and captured bytes are directly readable.
	upReq.Header.Set("Accept-Encoding", "identity")
	upReq.Host = hostOf(sess.Upstream)

	resp, err := p.client.Do(upReq)
	if err != nil {
		http.Error(w, "agenttape: upstream error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	respBody := streamCopy(w, resp.Body)
	completed := time.Now().UTC()

	// Keep this session's real request headers in memory (auth included) so the
	// exchange can be replayed to upstream later. Memory only — never persisted.
	p.sessions.RememberHeaders(sess.ID, r.Header)

	p.emit(p.buildEvent(sess, r, upURL, reqBody, resp, respBody, started, completed))
}

// buildEvent assembles a SourceEvent from the captured exchange.
func (p *Proxy) buildEvent(sess *Session, r *http.Request, upURL string, reqBody []byte, resp *http.Response, respBody []byte, started, completed time.Time) *event.SourceEvent {
	id := source.RandomID()
	ev := event.New(id, event.KindHTTPExchange, event.SourceRef{
		Kind:    event.SourceCapture,
		Adapter: p.Name(),
		Mode:    "reverse",
		Client:  sess.Client,
	})
	ev.Correlation = event.Correlation{SessionID: sess.ID}
	ev.Timing = event.Timing{
		StartedAt:   started.Format(time.RFC3339Nano),
		CompletedAt: completed.Format(time.RFC3339Nano),
		DurationMS:  completed.Sub(started).Milliseconds(),
	}

	reqArt := event.NewRawArtifact(id+":req", event.RoleRequestBody, reqBody)
	reqArt.MediaType = r.Header.Get("Content-Type")
	respArt := event.NewRawArtifact(id+":resp", event.RoleResponseBody, respBody)
	respArt.MediaType = resp.Header.Get("Content-Type")
	ev.RawArtifacts = []event.RawArtifact{reqArt, respArt}

	ev.Capture = &event.CaptureEvent{
		Protocol: "http",
		Method:   r.Method,
		URL:      r.URL.String(),
		Target:   upURL,
		Request: event.HTTPMessage{
			Headers:        redactHeaders(r.Header),
			ContentType:    r.Header.Get("Content-Type"),
			BodyArtifactID: reqArt.ID,
			BodySizeBytes:  int64(len(reqBody)),
		},
		Response: event.HTTPMessage{
			Headers:        redactHeaders(resp.Header),
			StatusCode:     resp.StatusCode,
			ContentType:    resp.Header.Get("Content-Type"),
			BodyArtifactID: respArt.ID,
			BodySizeBytes:  int64(len(respBody)),
		},
	}
	return &ev
}

// streamCopy writes src to dst, flushing as data arrives (so streamed tokens
// reach the agent live), while returning a full copy of the bytes for capture.
func streamCopy(dst http.ResponseWriter, src io.Reader) []byte {
	var buf bytes.Buffer
	flusher, _ := dst.(http.Flusher)
	chunk := make([]byte, 16*1024)
	for {
		n, err := src.Read(chunk)
		if n > 0 {
			_, _ = dst.Write(chunk[:n])
			buf.Write(chunk[:n])
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err != nil {
			break
		}
	}
	return buf.Bytes()
}

func splitToken(path string) (token, rest string) {
	path = strings.TrimPrefix(path, "/")
	if i := strings.IndexByte(path, '/'); i >= 0 {
		return path[:i], path[i+1:]
	}
	return path, ""
}

func hostOf(upstream string) string {
	u := strings.TrimPrefix(strings.TrimPrefix(upstream, "https://"), "http://")
	if i := strings.IndexByte(u, '/'); i >= 0 {
		u = u[:i]
	}
	return u
}
