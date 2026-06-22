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
	s.mux.HandleFunc("GET /api/terminals", s.handleTerminals)
	// In-memory sessions = the ones still replayable in this process (have creds).
	s.mux.HandleFunc("GET /api/active-sessions", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, s.Sessions.List())
	})
	s.mux.HandleFunc("GET /api/cases", func(w http.ResponseWriter, r *http.Request) { s.handleListCases(st, w, r) })
	s.mux.HandleFunc("POST /api/cases", func(w http.ResponseWriter, r *http.Request) { s.handleAddCase(st, w, r) })
	s.mux.HandleFunc("POST /api/cases/{id}/run", func(w http.ResponseWriter, r *http.Request) { s.handleRunCase(st, w, r) })
	s.mux.HandleFunc("GET /api/sessions/{id}/events", func(w http.ResponseWriter, r *http.Request) {
		events, err := st.ListEvents(r.PathValue("id"))
		if err != nil {
			httpError(w, err)
			return
		}
		writeJSON(w, events)
	})
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
