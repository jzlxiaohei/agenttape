package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var launchKinds = map[string]bool{"cc": true, "codex": true}

// sameOriginOK rejects cross-site callers (CSRF / DNS-rebinding) from triggering
// local process execution. Browsers always send Origin on POST; a non-browser
// caller (curl) sends none and is allowed.
func sameOriginOK(r *http.Request) bool {
	o := r.Header.Get("Origin")
	if o == "" {
		return true
	}
	u, err := url.Parse(o)
	if err != nil {
		return false
	}
	switch u.Hostname() {
	case "127.0.0.1", "localhost", "::1":
		return true
	}
	return false
}

// manualCommand is the FULL-CAPTURE copy-paste command (http + hooks): it runs the
// agent through `tracelab launch`, which injects hooks too. In key mode the key
// stays in the user's own shell env and never reaches tracelab's server. Extra
// args (e.g. --resume) are forwarded to the underlying client after `--`.
func manualCommand(exe, wd, serverURL, kind, mode, args string) string {
	prefix, suffix := "", ""
	if mode == "key" {
		if kind == "cc" {
			prefix = "ANTHROPIC_API_KEY=<YOUR_KEY> "
		} else {
			prefix = "OPENAI_API_KEY=<YOUR_KEY> "
			// codex defaults to the ChatGPT backend (subscription); an API key needs
			// the platform API instead, so pin the upstream explicitly.
			suffix = " -upstream https://api.openai.com/v1"
		}
	}
	if a := strings.TrimSpace(args); a != "" {
		// `--` stops tracelab's flag parser; everything after is handed to the client.
		suffix += " -- " + a
	}
	return fmt.Sprintf("cd %s && %s%s launch -kind %s -server %s%s",
		shellQuote(wd), prefix, shellQuote(exe), kind, shellQuote(serverURL), suffix)
}

// buildManualEnvCommand is the LIGHTWEIGHT "run it yourself" command: point the
// native client at the per-session proxy and run it directly. HTTP-only (no hooks),
// but the user keeps full control — any native flag (--resume, -m, …) can be
// appended. Only cc has a clean base-url env var; codex needs `-c` provider
// overrides. token may be a "<TOKEN>" placeholder for display before a session is
// registered.
func buildManualEnvCommand(kind, mode, serverURL, token, args string) string {
	base := strings.TrimRight(serverURL, "/") + "/s/" + token
	extra := strings.TrimSpace(args)
	var b strings.Builder

	if kind == "cc" {
		b.WriteString("export ANTHROPIC_BASE_URL=" + shellQuote(base) + "\n")
		if mode == "key" {
			b.WriteString("export ANTHROPIC_API_KEY='<YOUR_KEY>'\n")
		}
		b.WriteString("claude")
		if extra != "" {
			b.WriteString(" " + extra)
		}
		return b.String()
	}

	// codex: no base-url env var — route via `-c model_providers.*`. Each `-c` value
	// is single-quoted so the inner TOML double-quotes survive the shell.
	if mode == "key" {
		b.WriteString("export OPENAI_API_KEY='<YOUR_KEY>'\n")
	}
	b.WriteString("codex \\\n")
	b.WriteString("  -c 'model_provider=\"tracelab\"' \\\n")
	b.WriteString("  -c 'model_providers.tracelab.name=\"tracelab\"' \\\n")
	b.WriteString("  -c 'model_providers.tracelab.base_url=\"" + base + "\"' \\\n")
	b.WriteString("  -c 'model_providers.tracelab.wire_api=\"responses\"' \\\n")
	b.WriteString("  -c 'model_providers.tracelab.requires_openai_auth=true'")
	if extra != "" {
		b.WriteString(" " + extra)
	}
	return b.String()
}

// handleManualCommand returns the lightweight env/-c "run it yourself" command for
// a client. With register=true it first registers a session (so the proxy will
// accept the baked-in token) and returns its id; otherwise it returns a command
// with a "<TOKEN>" placeholder for display. No credentials are injected — in key
// mode the user supplies the key in their own shell, which the proxy forwards.
func (s *Server) handleManualCommand(w http.ResponseWriter, r *http.Request) {
	if !sameOriginOK(r) {
		http.Error(w, "cross-origin blocked", http.StatusForbidden)
		return
	}
	var req struct {
		Kind     string `json:"kind"`
		Mode     string `json:"mode"`
		Args     string `json:"args"`
		Upstream string `json:"upstream"`
		Register bool   `json:"register"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !launchKinds[req.Kind] {
		http.Error(w, "invalid request (kind must be cc | codex)", http.StatusBadRequest)
		return
	}
	serverURL := "http://" + r.Host
	token, sessionID := "<TOKEN>", ""
	if req.Register {
		upstream := req.Upstream
		if upstream == "" {
			upstream = defaultUpstream(req.Kind, req.Mode)
		}
		client := "codex_cli"
		if req.Kind == "cc" {
			client = "claude_code"
		}
		sess := s.Sessions.Register(client, upstream)
		token, sessionID = sess.Token, sess.ID
	}
	writeJSON(w, map[string]any{
		"command":    buildManualEnvCommand(req.Kind, req.Mode, serverURL, token, req.Args),
		"session_id": sessionID,
	})
}

// handleLaunch starts a coding agent in a NEW terminal window (so its TUI has a
// real tty) routed through this proxy, by running the existing `tracelab launch`
// CLI. Two credential modes:
//   - subscription: the agent uses the account you already logged in (no key).
//   - key: we register the session, keep the API key ONLY in process memory, and
//     the proxy injects it on forward — the agent is given a placeholder, so the
//     real key never reaches the agent, the terminal, or disk.
//
// MVP scope: macOS. codex desktop (global-config rewrite) is a separate endpoint.
func (s *Server) handleLaunch(w http.ResponseWriter, r *http.Request) {
	if !sameOriginOK(r) {
		http.Error(w, "cross-origin launch blocked", http.StatusForbidden)
		return
	}
	var req struct {
		Kind     string `json:"kind"`
		Workdir  string `json:"workdir"`
		Mode     string `json:"mode"`     // "subscription" (default) | "key"
		Upstream string `json:"upstream"` // optional override
		APIKey   string `json:"api_key"`
		Terminal string `json:"terminal"` // terminal app name (default Terminal)
		Args     string `json:"args"`     // extra native client args, forwarded after `--`
		Preview  bool   `json:"preview"`  // return the command without running anything
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !launchKinds[req.Kind] {
		http.Error(w, "invalid launch request (kind must be cc | codex)", http.StatusBadRequest)
		return
	}
	exe, err := os.Executable()
	if err != nil {
		httpError(w, err)
		return
	}
	wd := req.Workdir
	if wd == "" {
		if home, err := os.UserHomeDir(); err == nil {
			wd = home
		}
	}
	serverURL := "http://" + r.Host
	upstream := req.Upstream
	if upstream == "" {
		upstream = defaultUpstream(req.Kind, req.Mode)
	}

	// Preview: just hand back the copy-paste command (no exec, no session). Always
	// available, even when server launch is disabled.
	if req.Preview {
		writeJSON(w, map[string]any{"command": manualCommand(exe, wd, serverURL, req.Kind, req.Mode, req.Args), "enabled": s.AllowLaunch})
		return
	}

	// Past here we actually spawn a process — gated, OS-checked, and dir-validated.
	if !s.AllowLaunch {
		http.Error(w, "server launch is disabled; copy the command and run it yourself, or start serve with -allow-launch", http.StatusForbidden)
		return
	}
	if runtime.GOOS != "darwin" {
		http.Error(w, "web launch currently supports macOS only; use the CLI: tracelab launch -kind "+req.Kind, http.StatusNotImplemented)
		return
	}
	if req.Workdir != "" {
		if st, err := os.Stat(req.Workdir); err != nil || !st.IsDir() {
			http.Error(w, "working directory does not exist: "+req.Workdir, http.StatusBadRequest)
			return
		}
	}

	// Base launch command; key mode pre-registers a session (with proxy-injected
	// auth) and a placeholder key env so the real key never leaves this process.
	cmd := fmt.Sprintf("%s launch -kind %s -server %s", shellQuote(exe), req.Kind, shellQuote(serverURL))
	env := ""
	if req.Mode == "key" {
		if req.APIKey == "" {
			http.Error(w, "api_key required for key mode", http.StatusBadRequest)
			return
		}
		client := "codex_cli"
		if req.Kind == "cc" {
			client = "claude_code"
		}
		sess := s.Sessions.Register(client, upstream)
		s.Sessions.RememberInject(sess.ID, providerAuth(upstream, req.APIKey))
		cmd += fmt.Sprintf(" -token %s -session %s", shellQuote(sess.Token), shellQuote(sess.ID))
		if req.Kind == "cc" {
			env = "export ANTHROPIC_API_KEY=tracelab-proxy-placeholder\n"
		} else {
			env = "export OPENAI_API_KEY=tracelab-proxy-placeholder\n"
		}
	}
	if a := strings.TrimSpace(req.Args); a != "" {
		// `--` stops tracelab's flag parser; the rest is forwarded to the client. The
		// args run in the user's own local shell (their machine), like the workdir.
		cmd += " -- " + a
	}

	script := fmt.Sprintf("#!/bin/bash\n%scd %s || exit 1\nexec %s\n", env, shellQuote(wd), cmd)
	f, err := os.CreateTemp("", "tracelab-launch-*.sh")
	if err != nil {
		httpError(w, err)
		return
	}
	_, _ = f.WriteString(script)
	_ = f.Close()
	_ = os.Chmod(f.Name(), 0o700)

	term := req.Terminal
	if term == "" {
		term = "Terminal"
	}
	// open -a <App> <script>: exec args (not a shell), so the app name is safe.
	if err := exec.Command("open", "-a", term, f.Name()).Run(); err != nil {
		http.Error(w, "open "+term+": "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

// handleTerminals lists installed terminal apps the user can launch into.
func (s *Server) handleTerminals(w http.ResponseWriter, _ *http.Request) {
	if runtime.GOOS != "darwin" {
		writeJSON(w, []string{})
		return
	}
	writeJSON(w, detectTerminals())
}

func detectTerminals() []string {
	candidates := []string{"Terminal", "iTerm", "Ghostty", "WezTerm", "Alacritty", "kitty", "Warp", "Hyper"}
	out := []string{}
	for _, name := range candidates {
		if name == "Terminal" || appInstalled(name) { // Terminal.app ships with macOS
			out = append(out, name)
		}
	}
	return out
}

func appInstalled(name string) bool {
	paths := []string{"/Applications/" + name + ".app"}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, home+"/Applications/"+name+".app")
	}
	for _, p := range paths {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return true
		}
	}
	return false
}

// defaultUpstream picks the upstream by client and credential mode. codex splits
// by mode: a ChatGPT-subscription token only works against the Codex backend,
// while a real OpenAI API key only works against the platform API. Getting this
// wrong surfaces as a 401 "missing scopes: api.responses.write".
func defaultUpstream(kind, mode string) string {
	if kind == "cc" {
		return "https://api.anthropic.com"
	}
	if mode == "key" {
		return "https://api.openai.com/v1"
	}
	return "https://chatgpt.com/backend-api/codex"
}

// providerAuth builds the auth headers to inject for an API key, by wire format.
func providerAuth(upstream, apiKey string) http.Header {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	if strings.Contains(upstream, "anthropic") {
		h.Set("x-api-key", apiKey)
		h.Set("anthropic-version", "2023-06-01")
	} else {
		h.Set("Authorization", "Bearer "+apiKey)
	}
	return h
}

// shellQuote single-quotes a value so it is safe to embed in the launch script.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
