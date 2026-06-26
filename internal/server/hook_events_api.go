package server

import (
	"encoding/json"
	"net/http"
	"slices"

	"github.com/jzlxiaohei/agenttape/internal/store"
)

// hook_events_api.go exposes the per-client hook event registry: the set of
// lifecycle/tool events agenttape wires when launching each coding agent. It is
// seeded with built-in defaults but user-editable, so a user can capture a
// newly-shipped Claude Code / Codex event (or mute a noisy one) from the UI
// without waiting for a agenttape release. The launcher reads the ENABLED rows at
// launch time; see store.ClientHookEvents.

// handleListHookEvents returns every configured event across all clients (the UI
// groups them; the launch CLI filters to enabled-for-its-client).
func (s *Server) handleListHookEvents(st *store.Store, w http.ResponseWriter, _ *http.Request) {
	defs, err := st.ListHookEvents()
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, defs)
}

// handleAddHookEvent registers a user-added event for a client (enabled). The
// event name is intentionally NOT validated against a known set: unknown events
// are harmlessly ignored by the runtime, so adding one ahead of a release is the
// whole point.
func (s *Server) handleAddHookEvent(st *store.Store, w http.ResponseWriter, r *http.Request) {
	var req struct {
		Client string `json:"client"`
		Event  string `json:"event"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Event == "" || !validHookClient(req.Client) {
		http.Error(w, "invalid request (client must be one of "+hookClientList()+", event required)", http.StatusBadRequest)
		return
	}
	if err := st.AddHookEvent(req.Client, req.Event); err != nil {
		httpError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleSetHookEventEnabled toggles whether an event is wired on launch — used to
// mute a default without deleting it (deletion would let a later upgrade silently
// re-add it).
func (s *Server) handleSetHookEventEnabled(st *store.Store, w http.ResponseWriter, r *http.Request) {
	var req struct {
		Client  string `json:"client"`
		Event   string `json:"event"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Event == "" || !validHookClient(req.Client) {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := st.SetHookEventEnabled(req.Client, req.Event, req.Enabled); err == store.ErrNoRows {
		http.Error(w, "event not found", http.StatusNotFound)
		return
	} else if err != nil {
		httpError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleDeleteHookEvent removes a user-added event. Built-in (seed) events are
// protected — disable them instead, mirroring how seed replay cases are handled.
func (s *Server) handleDeleteHookEvent(st *store.Store, w http.ResponseWriter, r *http.Request) {
	var req struct {
		Client string `json:"client"`
		Event  string `json:"event"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Event == "" || !validHookClient(req.Client) {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	defs, err := st.ListHookEvents()
	if err != nil {
		httpError(w, err)
		return
	}
	idx := slices.IndexFunc(defs, func(d store.HookEventDef) bool {
		return d.Client == req.Client && d.Event == req.Event
	})
	if idx < 0 {
		http.Error(w, "event not found", http.StatusNotFound)
		return
	}
	if defs[idx].Source == "seed" {
		http.Error(w, "built-in default events cannot be deleted; disable them instead", http.StatusForbidden)
		return
	}
	if err := st.DeleteHookEvent(req.Client, req.Event); err != nil {
		httpError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validHookClient(client string) bool {
	return slices.Contains(store.HookClients, client)
}

func hookClientList() string {
	out := ""
	for i, c := range store.HookClients {
		if i > 0 {
			out += " | "
		}
		out += c
	}
	return out
}
