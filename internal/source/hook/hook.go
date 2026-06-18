// Package hook is the harness-hook capture adapter. Coding-agent harnesses POST
// their lifecycle/tool events here; each becomes the same event.SourceEvent that
// the HTTP capture adapter produces, proving the collection layer is decoupled
// from how data arrives (next.md 1.2, 7.1).
package hook

import (
	"io"
	"net/http"

	"tracelab/internal/event"
	"tracelab/internal/source"
)

// Adapter receives hook posts and emits SourceEvents.
type Adapter struct {
	emit source.Emitter
}

// New builds a hook adapter.
func New(emit source.Emitter) *Adapter { return &Adapter{emit: emit} }

// Name identifies the adapter.
func (a *Adapter) Name() string { return "hook" }

// Handler returns the hook endpoint (mounted at /_hook).
func (a *Adapter) Handler() http.Handler { return a }

// ServeHTTP accepts POST /_hook with the raw hook payload as the body. The
// runtime, event name, and session are passed as query parameters so the
// payload itself stays verbatim.
//
//	POST /_hook?runtime=claude_code&event=PreToolUse&session=<id>
func (a *Adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	q := r.URL.Query()
	runtime := orDefault(q.Get("runtime"), "unknown")
	eventName := orDefault(q.Get("event"), "unknown")

	ev := event.NewHookEvent(source.RandomID(), runtime, eventName, payload)
	if sid := q.Get("session"); sid != "" {
		ev.Correlation.SessionID = sid
	}
	if a.emit != nil {
		a.emit(&ev)
	}
	w.WriteHeader(http.StatusNoContent)
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
