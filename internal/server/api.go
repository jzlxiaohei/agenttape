package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"tracelab/internal/store"
)

// EnableAPI mounts the read-only viewer API backed by the store. It is only
// wired when serving with a SQLite store (the viewer needs persisted data).
func (s *Server) EnableAPI(st *store.Store) {
	// Re-attach proxy sessions persisted by an earlier run, so a still-running agent
	// keeps routing through the proxy after a restart. Only non-secret routing comes
	// back; key-mode sessions report NeedsKey() until the key is re-supplied.
	s.Sessions.BindPersister(&liveSessionPersister{st: st})

	s.mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, _ *http.Request) {
		sessions, err := st.ListSessions()
		if err != nil {
			httpError(w, err)
			return
		}
		writeJSON(w, sessions)
	})
	s.mux.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		results, err := st.SearchEvents(store.SearchFilters{
			Query:    q.Get("q"),
			Tag:      q.Get("tag"),
			Provider: q.Get("provider"),
			Client:   q.Get("client"),
		})
		if err != nil {
			httpError(w, err)
			return
		}
		writeJSON(w, results)
	})
	s.mux.HandleFunc("/api/facets", func(w http.ResponseWriter, _ *http.Request) {
		providers, clients, tags, err := st.Facets()
		if err != nil {
			httpError(w, err)
			return
		}
		writeJSON(w, map[string][]string{"providers": providers, "clients": clients, "tags": tags})
	})
	s.mux.HandleFunc("/api/stats", func(w http.ResponseWriter, _ *http.Request) {
		counts, err := st.TagCounts()
		if err != nil {
			httpError(w, err)
			return
		}
		writeJSON(w, map[string]any{"tags": counts})
	})
	s.mux.HandleFunc("POST /api/events/{id}/replay", func(w http.ResponseWriter, r *http.Request) {
		s.handleReplay(st, w, r)
	})
	s.mux.HandleFunc("POST /api/launch", s.handleLaunch)
	s.mux.HandleFunc("POST /api/launch/manual", s.handleManualCommand)
	s.mux.HandleFunc("GET /api/terminals", s.handleTerminals)
	s.mux.HandleFunc("GET /api/codex-desktop/status", func(w http.ResponseWriter, r *http.Request) { s.handleCodexDesktopStatus(w, r) })
	s.mux.HandleFunc("POST /api/codex-desktop/install", func(w http.ResponseWriter, r *http.Request) { s.handleCodexDesktopInstall(st, w, r) })
	s.mux.HandleFunc("POST /api/codex-desktop/restore", s.handleCodexDesktopRestore)
	// In-memory sessions = the ones still replayable in this process (have creds).
	s.mux.HandleFunc("GET /api/active-sessions", func(w http.ResponseWriter, _ *http.Request) {
		// credential_kind lets the replay UI tell apart same-client sessions that
		// carry genuinely different auth: "key" = a proxy-injected API key, else
		// "subscription" = captured login headers (same derivation as case curl).
		type dto struct {
			ID             string `json:"id"`
			Client         string `json:"client"`
			Upstream       string `json:"upstream"`
			Provider       string `json:"provider"`
			CredentialKind string `json:"credential_kind"`
			// NeedsKey: a key-mode session restored after a restart whose real key was
			// never persisted. It routes but would 401 until the key is re-supplied via
			// POST /api/active-sessions/{id}/key.
			NeedsKey bool `json:"needs_key"`
		}
		list := s.Sessions.List()
		out := make([]dto, 0, len(list))
		for _, ss := range list {
			kind := "subscription"
			if ss.Mode == "key" || s.Sessions.InjectFor(ss.ID) != nil {
				kind = "key"
			}
			out = append(out, dto{
				ID: ss.ID, Client: ss.Client, Upstream: ss.Upstream,
				Provider: ss.Provider, CredentialKind: kind,
				NeedsKey: s.Sessions.NeedsKey(ss.ID),
			})
		}
		writeJSON(w, out)
	})
	// Re-supply the API key for a key-mode session that lost it on restart. The key
	// goes ONLY into process memory (inject) — same as launch, never to disk. The
	// still-running agent (sending a placeholder) resumes on its next request.
	s.mux.HandleFunc("POST /api/active-sessions/{id}/key", func(w http.ResponseWriter, r *http.Request) {
		s.handleReinjectKey(w, r)
	})
	// Close a live session: forget it from the in-memory proxy registry (and drop
	// its creds). Does not kill the agent process — see Sessions.Remove.
	s.mux.HandleFunc("DELETE /api/active-sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		if !s.Sessions.Remove(r.PathValue("id")) {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	s.mux.HandleFunc("GET /api/cases", func(w http.ResponseWriter, r *http.Request) { s.handleListCases(st, w, r) })
	s.mux.HandleFunc("POST /api/cases", func(w http.ResponseWriter, r *http.Request) { s.handleAddCase(st, w, r) })
	s.mux.HandleFunc("POST /api/cases/{id}/run", func(w http.ResponseWriter, r *http.Request) { s.handleRunCase(st, w, r) })
	s.mux.HandleFunc("POST /api/cases/{id}/snapshot", func(w http.ResponseWriter, r *http.Request) { s.handleSnapshotCase(st, w, r) })
	s.mux.HandleFunc("POST /api/cases/{id}/overwrite", func(w http.ResponseWriter, r *http.Request) { s.handleOverwriteCase(st, w, r) })
	s.mux.HandleFunc("POST /api/cases/{id}/curl", func(w http.ResponseWriter, r *http.Request) { s.handleCaseCurl(st, w, r) })
	s.mux.HandleFunc("DELETE /api/cases/{id}", func(w http.ResponseWriter, r *http.Request) { s.handleDeleteCase(st, w, r) })
	s.mux.HandleFunc("GET /api/hook-events", func(w http.ResponseWriter, r *http.Request) { s.handleListHookEvents(st, w, r) })
	s.mux.HandleFunc("POST /api/hook-events", func(w http.ResponseWriter, r *http.Request) { s.handleAddHookEvent(st, w, r) })
	s.mux.HandleFunc("PATCH /api/hook-events", func(w http.ResponseWriter, r *http.Request) { s.handleSetHookEventEnabled(st, w, r) })
	s.mux.HandleFunc("DELETE /api/hook-events", func(w http.ResponseWriter, r *http.Request) { s.handleDeleteHookEvent(st, w, r) })
	s.mux.HandleFunc("GET /api/sessions/{id}/compaction-episodes", func(w http.ResponseWriter, r *http.Request) {
		comps, hooks, err := st.CompactionInputs(r.PathValue("id"))
		if err != nil {
			httpError(w, err)
			return
		}
		eps := detectCompactionEpisodes(comps, hooks)
		if eps == nil {
			eps = []CompactionEpisode{}
		}
		writeJSON(w, eps)
	})
	s.mux.HandleFunc("GET /api/sessions/{id}/events", func(w http.ResponseWriter, r *http.Request) {
		events, err := st.ListEvents(r.PathValue("id"))
		if err != nil {
			httpError(w, err)
			return
		}
		writeJSON(w, events)
	})
	s.mux.HandleFunc("PATCH /api/sessions/{id}", func(w http.ResponseWriter, r *http.Request) { s.handleRenameSession(st, w, r) })
	s.mux.HandleFunc("DELETE /api/sessions/{id}", func(w http.ResponseWriter, r *http.Request) { s.handleDeleteSession(st, w, r) })
	s.mux.HandleFunc("GET /api/events/{id}", func(w http.ResponseWriter, r *http.Request) {
		detail, err := st.GetEvent(r.PathValue("id"))
		if err == store.ErrNoRows {
			http.Error(w, "event not found", http.StatusNotFound)
			return
		}
		if err != nil {
			httpError(w, err)
			return
		}
		writeJSON(w, detail)
	})
	s.mux.HandleFunc("GET /api/events/{id}/raw/{role}", func(w http.ResponseWriter, r *http.Request) {
		path, err := st.RawFilePath(r.PathValue("id"), r.PathValue("role"))
		if err == store.ErrNoRows {
			http.Error(w, "raw file not found", http.StatusNotFound)
			return
		}
		if err != nil {
			httpError(w, err)
			return
		}
		http.ServeFile(w, r, path)
	})
}

// EnableViewer serves a built frontend (Vite dist) at /viewer if the directory
// exists. SPA routes fall back to index.html.
func (s *Server) EnableViewer(distDir string) bool {
	if st, err := os.Stat(distDir); err != nil || !st.IsDir() {
		return false
	}
	fs := http.FileServer(http.Dir(distDir))
	index := filepath.Join(distDir, "index.html")
	s.mux.HandleFunc("/viewer/", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/viewer")
		// SPA fallback: serve real files (assets) as-is, but route paths like
		// /sessions/:id — which have no file on disk — get index.html so client
		// routing can take over on deep links and reloads.
		rel := strings.TrimPrefix(filepath.Clean(r.URL.Path), "/")
		if rel == "" || rel == "." {
			http.ServeFile(w, r, index)
			return
		}
		if st, err := os.Stat(filepath.Join(distDir, rel)); err != nil || st.IsDir() {
			http.ServeFile(w, r, index)
			return
		}
		fs.ServeHTTP(w, r)
	})
	s.mux.HandleFunc("/viewer", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/viewer/", http.StatusFound)
	})
	return true
}

func httpError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
