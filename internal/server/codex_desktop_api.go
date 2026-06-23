package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"tracelab/internal/launcher"
	"tracelab/internal/store"
)

// codexDesktopState records an active injection so the UI can show "restore" across
// reloads and restore knows exactly what to undo. Lives next to the config it
// guards (~/.codex/), never in tracelab's data dir.
type codexDesktopState struct {
	HadOriginal bool   `json:"had_original"`
	SessionID   string `json:"session_id"`
	ConfigPath  string `json:"config_path"`
	BackupPath  string `json:"backup_path"`
	InstalledAt string `json:"installed_at"`
	Hooks       bool   `json:"hooks"`
}

func codexPaths() (dir, config, backup, state string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", "", "", err
	}
	dir = filepath.Join(home, ".codex")
	return dir,
		filepath.Join(dir, "config.toml"),
		filepath.Join(dir, "config.toml.tracelab-bak"),
		filepath.Join(dir, ".tracelab-desktop-state.json"),
		nil
}

func readCodexState(statePath string) (*codexDesktopState, bool) {
	b, err := os.ReadFile(statePath)
	if err != nil {
		return nil, false
	}
	var st codexDesktopState
	if json.Unmarshal(b, &st) != nil {
		return nil, false
	}
	return &st, true
}

// handleCodexDesktopStatus reports whether tracelab routing is currently injected
// into ~/.codex/config.toml (so the UI can offer restore after a reload).
func (s *Server) handleCodexDesktopStatus(w http.ResponseWriter, _ *http.Request) {
	_, _, _, statePath, err := codexPaths()
	if err != nil {
		httpError(w, err)
		return
	}
	st, ok := readCodexState(statePath)
	if !ok {
		writeJSON(w, map[string]any{"active": false})
		return
	}
	writeJSON(w, map[string]any{
		"active":       true,
		"session_id":   st.SessionID,
		"config_path":  st.ConfigPath,
		"installed_at": st.InstalledAt,
		"hooks":        st.Hooks,
	})
}

// handleCodexDesktopInstall backs up ~/.codex/config.toml verbatim and writes a
// version that routes the desktop app through this proxy (subscription auth) plus
// optional hook capture. Mutating global config breaks the non-invasive rule, so
// it is gated on -allow-launch and always paired with the verbatim backup + the
// restore endpoint. Subscription only — the desktop app reads ~/.codex/auth.json,
// which we never touch.
func (s *Server) handleCodexDesktopInstall(st *store.Store, w http.ResponseWriter, r *http.Request) {
	if !sameOriginOK(r) {
		http.Error(w, "cross-origin blocked", http.StatusForbidden)
		return
	}
	if !s.AllowLaunch {
		http.Error(w, "server launch is disabled; restart serve with -allow-launch to enable Codex desktop routing", http.StatusForbidden)
		return
	}
	var req struct {
		Hooks    *bool  `json:"hooks"` // default true
		Upstream string `json:"upstream"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	withHooks := req.Hooks == nil || *req.Hooks

	dir, configPath, backupPath, statePath, err := codexPaths()
	if err != nil {
		httpError(w, err)
		return
	}
	// Never stack injections — require an explicit restore first.
	if _, active := readCodexState(statePath); active {
		http.Error(w, "tracelab routing is already active; restore it first", http.StatusConflict)
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		httpError(w, err)
		return
	}

	original, readErr := os.ReadFile(configPath)
	hadOriginal := readErr == nil
	if hadOriginal && launcher.HasCodexMarker(string(original)) {
		http.Error(w, "config.toml already has a tracelab block but no state file; remove it manually, then retry", http.StatusConflict)
		return
	}

	upstream := req.Upstream
	if upstream == "" {
		upstream = defaultUpstream("codex", "subscription")
	}
	// Hook capture uses the user-editable enabled set from the store, identical to
	// what the CLI/desktop launch injects. If the user disabled every codex event,
	// no hook block is written even when hooks were requested.
	var events []string
	if withHooks {
		if events, err = st.ClientHookEvents("codex"); err != nil {
			httpError(w, err)
			return
		}
	}
	hooksInjected := len(events) > 0

	sess := s.Sessions.Register("codex_cli", upstream)
	merged := launcher.MergeCodexDesktopConfig(string(original), "http://"+r.Host, sess, events)

	if hadOriginal {
		if err := os.WriteFile(backupPath, original, 0o600); err != nil {
			httpError(w, err)
			return
		}
	}
	if err := os.WriteFile(configPath, []byte(merged), 0o600); err != nil {
		httpError(w, err)
		return
	}
	state := codexDesktopState{
		HadOriginal: hadOriginal,
		SessionID:   sess.ID,
		ConfigPath:  configPath,
		BackupPath:  backupPath,
		InstalledAt: time.Now().UTC().Format(time.RFC3339),
		Hooks:       hooksInjected,
	}
	stb, _ := json.Marshal(state)
	_ = os.WriteFile(statePath, stb, 0o600)

	writeJSON(w, map[string]any{
		"ok":           true,
		"session_id":   sess.ID,
		"config_path":  configPath,
		"backup_path":  backupPath,
		"had_original": hadOriginal,
		"hooks":        hooksInjected,
	})
}

// handleCodexDesktopRestore puts the original config back byte-exact (or removes the
// file if there was none) and clears state. A no-op if nothing is active.
func (s *Server) handleCodexDesktopRestore(w http.ResponseWriter, r *http.Request) {
	if !sameOriginOK(r) {
		http.Error(w, "cross-origin blocked", http.StatusForbidden)
		return
	}
	_, configPath, _, statePath, err := codexPaths()
	if err != nil {
		httpError(w, err)
		return
	}
	st, ok := readCodexState(statePath)
	if !ok {
		writeJSON(w, map[string]any{"ok": true, "restored": false})
		return
	}
	if st.HadOriginal {
		backup, err := os.ReadFile(st.BackupPath)
		if err != nil {
			httpError(w, err)
			return
		}
		if err := os.WriteFile(configPath, backup, 0o600); err != nil {
			httpError(w, err)
			return
		}
		_ = os.Remove(st.BackupPath)
	} else {
		_ = os.Remove(configPath)
	}
	_ = os.Remove(statePath)
	writeJSON(w, map[string]any{"ok": true, "restored": true})
}
