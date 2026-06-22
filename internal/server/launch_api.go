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

// manualCommand is the copy-paste command a user can run themselves instead of
// letting the server spawn anything. In key mode the key stays in the user's own
// shell env and never reaches tracelab's server.
func manualCommand(exe, wd, serverURL, kind, mode string) string {
	prefix := ""
	if mode == "key" {
		if kind == "cc" {
			prefix = "ANTHROPIC_API_KEY=<YOUR_KEY> "
		} else {
			prefix = "OPENAI_API_KEY=<YOUR_KEY> "
		}
	}
	return fmt.Sprintf("cd %s && %s%s launch -kind %s -server %s",
		shellQuote(wd), prefix, shellQuote(exe), kind, shellQuote(serverURL))
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
		upstream = defaultUpstream(req.Kind)
	}

	// Preview: just hand back the copy-paste command (no exec, no session). Always
	// available, even when server launch is disabled.
	if req.Preview {
		writeJSON(w, map[string]any{"command": manualCommand(exe, wd, serverURL, req.Kind, req.Mode), "enabled": s.AllowLaunch})
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

func defaultUpstream(kind string) string {
	if kind == "cc" {
		return "https://api.anthropic.com"
	}
	return "https://api.openai.com/v1"
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
