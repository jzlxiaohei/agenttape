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
// agent through `agenttape launch`, which injects hooks too. In key mode the key
// stays in the user's own shell env and never reaches agenttape's server. Extra
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
		// `--` stops agenttape's flag parser; everything after is handed to the client.
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
	b.WriteString("  -c 'model_provider=\"agenttape\"' \\\n")
	b.WriteString("  -c 'model_providers.agenttape.name=\"agenttape\"' \\\n")
	b.WriteString("  -c 'model_providers.agenttape.base_url=\"" + base + "\"' \\\n")
	b.WriteString("  -c 'model_providers.agenttape.wire_api=\"responses\"' \\\n")
	b.WriteString("  -c 'model_providers.agenttape.requires_openai_auth=true'")
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
		mode := req.Mode
		if mode == "" {
			mode = "subscription"
		}
		spec, _ := agentProviderByKind(req.Kind) // kind validated above by launchKinds
		sess := s.Sessions.Register(spec.Client, upstream, spec.Provider, mode)
		token, sessionID = sess.Token, sess.ID
	}
	writeJSON(w, map[string]any{
		"command":    buildManualEnvCommand(req.Kind, req.Mode, serverURL, token, req.Args),
		"session_id": sessionID,
	})
}

// handleLaunch starts a coding agent in a NEW terminal window (so its TUI has a
// real tty) routed through this proxy, by running the existing `agenttape launch`
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
		http.Error(w, "web launch currently supports macOS only; use the CLI: agenttape launch -kind "+req.Kind, http.StatusNotImplemented)
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
		spec, ok := agentProviderByKind(req.Kind)
		if !ok {
			http.Error(w, "unknown provider kind: "+req.Kind, http.StatusBadRequest)
			return
		}
		sess := s.Sessions.Register(spec.Client, upstream, spec.Provider, "key")
		// The real key lives ONLY in process memory (inject); the agent is handed a
		// placeholder env so the key never reaches the agent, the terminal, or disk.
		s.Sessions.RememberInject(sess.ID, spec.injectAuth(req.APIKey))
		cmd += fmt.Sprintf(" -token %s -session %s", shellQuote(sess.Token), shellQuote(sess.ID))
		env = "export " + spec.KeyEnv + "=agenttape-proxy-placeholder\n"
	}
	if a := strings.TrimSpace(req.Args); a != "" {
		// `--` stops agenttape's flag parser; the rest is forwarded to the client. The
		// args run in the user's own local shell (their machine), like the workdir.
		cmd += " -- " + a
	}

	script := fmt.Sprintf("#!/bin/bash\n%scd %s || exit 1\nexec %s\n", env, shellQuote(wd), cmd)
	f, err := os.CreateTemp("", "agenttape-launch-*.sh")
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
	// The terminal name is free-text (the UI lets users type any installed app), so
	// validate it before launching — a typo otherwise surfaces as an opaque `open`
	// failure. appLaunchable resolves via Launch Services, the same lookup `open -a`
	// uses, so a pass here means the open below will find the app.
	if !appLaunchable(term) {
		http.Error(w, "terminal app not found: "+term+" — check the name (must match an installed .app), or install it", http.StatusBadRequest)
		return
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

// handleReinjectKey puts a real API key back into memory for a key-mode session that
// lost it on a restart (NeedsKey). The key goes ONLY into the in-memory inject map,
// exactly as the original launch did — never to disk. The still-running agent (which
// keeps sending the placeholder) resumes on its next request, transparently.
func (s *Server) handleReinjectKey(w http.ResponseWriter, r *http.Request) {
	if !sameOriginOK(r) {
		http.Error(w, "cross-origin blocked", http.StatusForbidden)
		return
	}
	sess := s.Sessions.Get(r.PathValue("id"))
	if sess == nil {
		http.Error(w, "session not live in this process", http.StatusNotFound)
		return
	}
	if sess.Mode != "key" {
		http.Error(w, "session is not an API-key session; nothing to re-supply", http.StatusBadRequest)
		return
	}
	var req struct {
		APIKey string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.APIKey) == "" {
		http.Error(w, "api_key required", http.StatusBadRequest)
		return
	}
	spec, ok := agentProviderByProvider(sess.Provider)
	if !ok {
		http.Error(w, "no provider spec for "+sess.Provider, http.StatusBadRequest)
		return
	}
	s.Sessions.RememberInject(sess.ID, spec.injectAuth(strings.TrimSpace(req.APIKey)))
	writeJSON(w, map[string]any{"ok": true})
}

func detectTerminals() []string {
	candidates := []string{"Terminal", "iTerm", "Ghostty", "WezTerm", "Alacritty", "kitty", "Warp", "Hyper"}
	out := []string{}
	for _, name := range candidates {
		if name == "Terminal" || appLaunchable(name) {
			out = append(out, name)
		}
	}
	return out
}

// appLaunchable reports whether macOS can resolve an app by the given name, using
// the same Launch Services lookup that `open -a` performs. It has no side effects
// — it neither launches the app nor reveals it in Finder. macOS only; returns
// false elsewhere.
func appLaunchable(name string) bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	// Embed the name into an AppleScript string literal; escape backslash and quote
	// so a name containing either can't break out of the literal.
	esc := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(name)
	script := `POSIX path of (path to application "` + esc + `")`
	return exec.Command("osascript", "-e", script).Run() == nil
}

// defaultUpstream picks the upstream by launch kind and credential mode, via the
// provider registry (agent_providers.go) — the single source for which host each
// provider's subscription token vs API key must hit. Unknown kinds fall back to the
// Anthropic host so callers always get a usable default.
func defaultUpstream(kind, mode string) string {
	if spec, ok := agentProviderByKind(kind); ok {
		return spec.upstreamFor(mode)
	}
	return agentProviders["cc"].SubURL
}

// shellQuote single-quotes a value so it is safe to embed in the launch script.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
