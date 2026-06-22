package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"tracelab/internal/launcher"
	"tracelab/internal/source/httpcap"
)

func runLaunch(args []string) error {
	fs := flag.NewFlagSet("launch", flag.ExitOnError)
	kind := fs.String("kind", "", "client kind: cc | codex")
	serverURL := fs.String("server", "http://127.0.0.1:8787", "running tracelab server")
	upstream := fs.String("upstream", "", "upstream base URL (defaults by kind)")
	token := fs.String("token", "", "use a pre-registered session token (skip register)")
	session := fs.String("session", "", "pre-registered session id (use with -token)")
	_ = fs.Parse(args)

	client, defUpstream, err := clientDefaults(*kind)
	if err != nil {
		return err
	}
	if *upstream == "" {
		*upstream = defUpstream
	}

	tok, sessionID := *token, *session
	if tok == "" {
		if tok, sessionID, err = register(*serverURL, client, *upstream); err != nil {
			return fmt.Errorf("register session: %w", err)
		}
	}
	sess := &httpcap.Session{ID: sessionID, Token: tok}

	cmd := chooseLauncher(*kind, *serverURL, sess, fs.Args())
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	fmt.Fprintf(os.Stderr, "tracelab: launching %s via %s (no global config modified)\n",
		client, httpcap.SessionBaseURL(*serverURL, sess))
	return cmd.Run()
}

func clientDefaults(kind string) (client, upstream string, err error) {
	switch kind {
	case "cc":
		return "claude_code", "https://api.anthropic.com", nil
	case "codex":
		return "codex_cli", "https://api.openai.com/v1", nil
	default:
		return "", "", fmt.Errorf("unknown -kind %q (want cc | codex)", kind)
	}
}

func chooseLauncher(kind, serverURL string, sess *httpcap.Session, args []string) *exec.Cmd {
	if kind == "codex" {
		return launcher.LaunchCodex(serverURL, sess, args...)
	}
	return launcher.LaunchClaudeCode(serverURL, sess, args...)
}

func register(serverURL, client, upstream string) (token, sessionID string, err error) {
	body, _ := json.Marshal(map[string]string{"client": client, "upstream": upstream})
	resp, err := http.Post(serverURL+"/_register", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("server returned %d", resp.StatusCode)
	}
	var out struct {
		Token     string `json:"token"`
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", err
	}
	return out.Token, out.SessionID, nil
}
