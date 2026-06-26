package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"agenttape/internal/launcher"
	"agenttape/internal/source/hook"
	"agenttape/internal/source/httpcap"
)

func runLaunch(args []string) error {
	fs := flag.NewFlagSet("launch", flag.ExitOnError)
	kind := fs.String("kind", "", "client kind: cc | codex")
	serverURL := fs.String("server", "http://127.0.0.1:8787", "running agenttape server")
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

	// The hook event set is user-editable in the store; pull the enabled events
	// from the running server so a launch honors live config without rebuilding.
	// If the server can't answer (older server, no store), fall back to the
	// built-in defaults so capture still works.
	events := fetchHookEvents(*serverURL, hookClient(*kind))

	cmd := chooseLauncher(*kind, *serverURL, sess, events, fs.Args())
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	fmt.Fprintf(os.Stderr, "agenttape: launching %s via %s (no global config modified)\n",
		client, httpcap.SessionBaseURL(*serverURL, sess))
	return cmd.Run()
}

func clientDefaults(kind string) (client, upstream string, err error) {
	switch kind {
	case "cc":
		return "claude_code", "https://api.anthropic.com", nil
	case "codex":
		// Subscription (codex login / ChatGPT) is the common CLI path: its token is
		// scoped for the Codex backend, NOT the platform API. Routing to
		// api.openai.com yields a 401 "missing scopes: api.responses.write". API-key
		// users must pass -upstream https://api.openai.com/v1 explicitly.
		return "codex_cli", "https://chatgpt.com/backend-api/codex", nil
	default:
		return "", "", fmt.Errorf("unknown -kind %q (want cc | codex)", kind)
	}
}

func chooseLauncher(kind, serverURL string, sess *httpcap.Session, events, args []string) *exec.Cmd {
	if kind == "codex" {
		return launcher.LaunchCodex(serverURL, sess, events, args...)
	}
	return launcher.LaunchClaudeCode(serverURL, sess, events, args...)
}

// hookClient maps a launch kind to the client key used in the hook event store
// (both codex CLI and desktop share "codex").
func hookClient(kind string) string {
	if kind == "codex" {
		return "codex"
	}
	return "claude_code"
}

// fetchHookEvents asks the server for the user's enabled hook events for a
// client. On any failure it falls back to the built-in defaults so a launch
// never silently captures nothing because the registry endpoint was unavailable.
func fetchHookEvents(serverURL, client string) []string {
	fallback := func() []string {
		if client == "codex" {
			return hook.DefaultCodexEvents()
		}
		return hook.DefaultClaudeEvents()
	}
	resp, err := http.Get(serverURL + "/api/hook-events")
	if err != nil {
		return fallback()
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fallback()
	}
	var defs []struct {
		Client  string `json:"client"`
		Event   string `json:"event"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&defs); err != nil {
		return fallback()
	}
	out := []string{}
	for _, d := range defs {
		if d.Client == client && d.Enabled {
			out = append(out, d.Event)
		}
	}
	// An empty list is a legitimate "capture nothing" choice and is respected;
	// only a transport/parse failure (handled above) falls back to defaults.
	return out
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
