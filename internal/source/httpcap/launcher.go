package httpcap

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// SessionBaseURL is the per-session proxy entrypoint a client should be pointed
// at: <proxyBase>/s/<token>.
func SessionBaseURL(proxyBase string, sess *Session) string {
	return strings.TrimRight(proxyBase, "/") + "/s/" + sess.Token
}

// claudeHookEvents is the full set of Claude Code hook events we capture to
// study harness orchestration (next.md 7.1). Sourced from the official docs
// (code.claude.com/docs/en/hooks). We capture everything EXCEPT MessageDisplay,
// which fires continuously while assistant text streams and would flood capture.
// Unknown event names are harmlessly ignored by older Claude Code versions.
var claudeHookEvents = []string{
	// session lifecycle
	"SessionStart", "Setup", "SessionEnd",
	// per-turn
	"UserPromptSubmit", "UserPromptExpansion", "Stop", "StopFailure",
	// agentic loop / tools
	"PreToolUse", "PostToolUse", "PostToolUseFailure", "PostToolBatch",
	"PermissionRequest", "PermissionDenied",
	// subagents & teams
	"SubagentStart", "SubagentStop", "TeammateIdle",
	// task management
	"TaskCreated", "TaskCompleted",
	// context & configuration
	"InstructionsLoaded", "ConfigChange", "CwdChanged",
	// file & system
	"FileChanged", "WorktreeCreate", "WorktreeRemove",
	// compaction
	"PreCompact", "PostCompact",
	// notification & MCP
	"Notification", "Elicitation", "ElicitationResult",
}

// LaunchClaudeCode builds a command that runs Claude Code through the proxy
// (ANTHROPIC_BASE_URL) and injects hook capture via `--settings` with a JSON
// string. Both apply to this invocation only — NO global config or files are
// mutated (next.md 1.1). The hooks POST each event to the proxy's /_hook
// endpoint stamped with this session's id, so hook events correlate with the
// HTTP captures of the same session.
func LaunchClaudeCode(proxyBase string, sess *Session, args ...string) *exec.Cmd {
	settings := buildClaudeHookSettings(proxyBase, sess.ID)
	full := append([]string{"--settings", settings}, args...)
	cmd := exec.Command("claude", full...)
	cmd.Env = append(os.Environ(),
		"ANTHROPIC_BASE_URL="+SessionBaseURL(proxyBase, sess),
		"TRACELAB_SESSION="+sess.ID,
	)
	return cmd
}

// buildClaudeHookSettings returns a Claude Code settings JSON string wiring each
// lifecycle event to a curl that ships the hook payload to /_hook.
func buildClaudeHookSettings(proxyBase, sessionID string) string {
	type hookCmd struct {
		Type    string `json:"type"`
		Command string `json:"command"`
	}
	type hookGroup struct {
		Hooks []hookCmd `json:"hooks"`
	}
	hookURL := strings.TrimRight(proxyBase, "/") + "/_hook"

	hooks := map[string][]hookGroup{}
	for _, ev := range claudeHookEvents {
		url := fmt.Sprintf("%s?runtime=claude_code&event=%s&session=%s", hookURL, ev, sessionID)
		cmd := fmt.Sprintf("curl -sS -m 2 -X POST %q --data-binary @- >/dev/null 2>&1", url)
		hooks[ev] = []hookGroup{{Hooks: []hookCmd{{Type: "command", Command: cmd}}}}
	}
	b, _ := json.Marshal(map[string]any{"hooks": hooks})
	return string(b)
}

// LaunchCodex builds a command that runs the Codex CLI through the proxy using
// single-run `-c` overrides. This is the clean, non-invasive alternative to
// editing ~/.codex/config.toml: the overrides apply to this invocation only and
// nothing on disk is changed (next.md 1.1).
//
// Codex Desktop App is intentionally NOT handled here — its background process
// does not reliably inherit `-c` overrides, and we refuse to silently rewrite
// the user's global config as a workaround. That limitation is documented rather
// than worked around.
func LaunchCodex(proxyBase string, sess *Session, args ...string) *exec.Cmd {
	base := SessionBaseURL(proxyBase, sess)
	overrides := []string{
		"-c", `model_provider="tracelab"`,
		"-c", `model_providers.tracelab.name="tracelab capture"`,
		"-c", `model_providers.tracelab.base_url="` + base + `"`,
		"-c", `model_providers.tracelab.wire_api="responses"`,
		"-c", `model_providers.tracelab.requires_openai_auth=true`,
	}
	cmd := exec.Command("codex", append(overrides, args...)...)
	cmd.Env = os.Environ()
	return cmd
}
