package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"tracelab/internal/normalize/providers"
	"tracelab/internal/server"
	"tracelab/internal/sink"
	"tracelab/internal/store"
)

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	listen := fs.String("listen", "127.0.0.1:8787", "listen address")
	data := fs.String("data", "tracelab-data", "data dir (SQLite db + raw files)")
	jsonlOut := fs.String("jsonl", "", "use a JSONL file instead of SQLite (debug)")
	viewer := fs.String("viewer", "frontend/dist", "built viewer dist dir to serve at /viewer")
	_ = fs.Parse(args)

	var s sink.Sink
	var where string
	var st *store.Store
	if *jsonlOut != "" {
		js, err := sink.NewJSONL(*jsonlOut)
		if err != nil {
			return fmt.Errorf("open jsonl sink: %w", err)
		}
		s, where = js, *jsonlOut
	} else {
		opened, err := store.Open(*data)
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}
		st, s, where = opened, opened, *data+"/ (tracelab.db + raw/)"
	}
	defer s.Close()

	srv := server.New(s, providers.Registry())

	log.Printf("tracelab serve: listening on http://%s", *listen)
	log.Printf("  proxy:    http://%s/s/<token>/...", *listen)
	log.Printf("  hooks:    POST http://%s/_hook?runtime=..&event=..", *listen)
	log.Printf("  register: POST http://%s/_register {client,upstream}", *listen)
	log.Printf("  writing:  %s", where)
	if st != nil {
		srv.EnableAPI(st)
		log.Printf("  api:      http://%s/api/sessions", *listen)
		if srv.EnableViewer(*viewer) {
			log.Printf("  viewer:   http://%s/viewer/", *listen)
		}
	}
	return http.ListenAndServe(*listen, srv.Handler())
}
