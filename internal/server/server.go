// Package server wires the collection adapters, the normalize registry, and a
// sink into one HTTP service. It is the composition root for module 1; the
// layers it connects do not know about each other.
package server

import (
	"encoding/json"
	"log"
	"net/http"

	"tracelab/internal/event"
	"tracelab/internal/normalize"
	"tracelab/internal/sink"
	"tracelab/internal/source/hook"
	"tracelab/internal/source/httpcap"
)

// Server holds the shared session registry and exposes a single mux.
type Server struct {
	Sessions *httpcap.Sessions
	mux      *http.ServeMux
	sink     sink.Sink
	reg      *normalize.Registry
	// AllowLaunch gates the server-side "launch in a terminal" action. On by
	// default (serve sets it); pass -allow-launch=false to disable, after which the
	// server executes nothing. The Launch page still shows a copy-paste command
	// regardless.
	AllowLaunch bool
}

// New builds the server. emit normalizes each event and writes it to the sink.
func New(s sink.Sink, reg *normalize.Registry) *Server {
	srv := &Server{
		Sessions: httpcap.NewSessions(),
		mux:      http.NewServeMux(),
		sink:     s,
		reg:      reg,
	}
	emit := srv.emit

	proxy := httpcap.NewProxy(srv.Sessions, emit)
	hookA := hook.New(emit)

	srv.mux.Handle("/s/", proxy.Handler())
	srv.mux.Handle("/_hook", hookA.Handler())
	srv.mux.HandleFunc("/_register", srv.handleRegister)
	srv.mux.HandleFunc("/_sessions", srv.handleSessions)
	srv.mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	return srv
}

// Handler exposes the composed mux.
func (s *Server) Handler() http.Handler { return s.mux }

// emit normalizes (best-effort) and persists one event.
func (s *Server) emit(ev *event.SourceEvent) {
	rec := sink.Record{Event: ev}
	if env, err := s.reg.Normalize(ev); err != nil {
		rec.NormalizeError = err.Error()
	} else {
		rec.Normalized = env
	}
	if err := s.sink.Write(rec); err != nil {
		log.Printf("tracelab: sink write failed for event %s: %v", ev.ID, err)
	}
}

type registerReq struct {
	Client   string `json:"client"`
	Upstream string `json:"upstream"`
}

type registerResp struct {
	SessionID string `json:"session_id"`
	Token     string `json:"token"`
	BaseURL   string `json:"base_url"`
}

// handleRegister creates a capture session and returns the per-session base URL
// a launcher should point the client at.
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Upstream == "" {
		http.Error(w, "invalid register request", http.StatusBadRequest)
		return
	}
	sess := s.Sessions.Register(req.Client, req.Upstream)
	writeJSON(w, registerResp{
		SessionID: sess.ID,
		Token:     sess.Token,
		BaseURL:   httpcap.SessionBaseURL("http://"+r.Host, sess),
	})
}

func (s *Server) handleSessions(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, s.Sessions.List())
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
