// Command tracelab runs the module-1 capture service and helper subcommands.
//
//	tracelab serve   — run the capture proxy + hook endpoint
//	tracelab launch  — start cc/codex through the proxy (non-invasive)
//	tracelab dump    — inspect captured records (normalized view)
package main

import (
	"fmt"
	"os"
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
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "tracelab:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `tracelab — coding-agent request lab (module 1: capture + normalize)

usage:
  tracelab serve  [-listen 127.0.0.1:8787] [-out traces.jsonl]
  tracelab launch -kind cc|codex [-server http://127.0.0.1:8787] [-upstream URL] -- <client args>
  tracelab dump   <traces.jsonl>
`)
}
