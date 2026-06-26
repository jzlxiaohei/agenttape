// Command agenttape runs the module-1 capture service and helper subcommands.
//
//	agenttape serve   — run the capture proxy + hook endpoint
//	agenttape launch  — start cc/codex through the proxy (non-invasive)
//	agenttape dump    — inspect captured records (normalized view)
package main

import (
	"fmt"
	"os"
	"runtime"
)

// Build metadata, overridden via -ldflags "-X main.version=... -X main.commit=... -X main.date=...".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "serve":
		err = runServe(os.Args[2:])
	case "launch":
		err = runLaunch(os.Args[2:])
	case "dump":
		err = runDump(os.Args[2:])
	case "version", "--version", "-version", "-v":
		fmt.Printf("agenttape %s (commit %s, built %s, %s)\n",
			version, commit, date, runtime.Version())
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "agenttape:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `agenttape — coding-agent request lab (module 1: capture + normalize)

usage:
  agenttape serve  [-listen 127.0.0.1:8787] [-out traces.jsonl]
  agenttape launch -kind cc|codex [-server http://127.0.0.1:8787] [-upstream URL] -- <client args>
  agenttape dump   <traces.jsonl>
  agenttape version
`)
}
