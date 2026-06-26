package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"agenttape/internal/normalize/providers"
	"agenttape/internal/server"
	"agenttape/internal/sink"
	"agenttape/internal/store"
	"agenttape/internal/web"
)

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	listen := fs.String("listen", "127.0.0.1:8787", "listen address")
	data := fs.String("data", "agenttape-data", "data dir (SQLite db + raw files)")
	jsonlOut := fs.String("jsonl", "", "use a JSONL file instead of SQLite (debug)")
	viewer := fs.String("viewer", "", "serve the viewer from this dist dir instead of the embedded bundle (frontend dev only; empty = use the binary's embedded viewer)")
	allowLaunch := fs.Bool("allow-launch", true, "allow the viewer to launch agents in a terminal (on by default; pass -allow-launch=false to disable — the page always still shows a copy-paste command)")
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
		st, s, where = opened, opened, *data+"/ (agenttape.db + raw/)"
	}
	defer s.Close()

	srv := server.New(s, providers.Registry())
	srv.AllowLaunch = *allowLaunch

	log.Printf("agenttape serve: listening on http://%s", *listen)
	log.Printf("  proxy:    http://%s/s/<token>/...", *listen)
	log.Printf("  hooks:    POST http://%s/_hook?runtime=..&event=..", *listen)
	log.Printf("  register: POST http://%s/_register {client,upstream}", *listen)
	log.Printf("  writing:  %s", where)
	if st != nil {
		srv.EnableAPI(st)
		log.Printf("  api:      http://%s/api/sessions", *listen)
		// Default: the viewer embedded in this binary (single-file distribution).
		// -viewer <dir> overrides it with an on-disk dist for frontend dev.
		viewerFS := web.Dist()
		if *viewer != "" {
			viewerFS = os.DirFS(*viewer)
		}
		if srv.EnableViewer(viewerFS) {
			log.Printf("  viewer:   http://%s/viewer/", *listen)
		}
	}
	return http.ListenAndServe(*listen, srv.Handler())
}
