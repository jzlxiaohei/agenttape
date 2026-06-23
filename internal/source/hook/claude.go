package hook

import (
	"encoding/json"
	"fmt"
	"strings"
)

// claudeEvents is the BUILT-IN DEFAULT set of Claude Code hook events, used to
// seed the store's per-client registry on first run. It is not read at launch:
// the launcher reads the (user-editable) enabled set from the store, so a user
// can add an event the moment a new Claude Code release ships one — no tracelab
// release required. Sourced from the official docs (code.claude.com/docs/en/hooks).
// We default-capture everything EXCEPT MessageDisplay, which fires continuously
// while assistant text streams and would flood capture (a user can still add it).
// Unknown event names are harmlessly ignored by older Claude Code versions, so an
// added-but-not-yet-real event is safe.
var claudeEvents = []string{
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

// DefaultClaudeEvents returns a copy of the built-in default Claude Code event
// set, used to seed the store and as the launcher's fallback when the store is
// unreachable.
func DefaultClaudeEvents() []string {
	return append([]string(nil), claudeEvents...)
}

// BuildClaudeSettings returns a Claude Code settings JSON string wiring each of
// the given lifecycle events to a curl that ships the hook payload to /_hook.
func BuildClaudeSettings(events []string, baseURL, sessionID string) string {
	type hookCmd struct {
		Type    string `json:"type"`
		Command string `json:"command"`
	}
	type hookGroup struct {
		Hooks []hookCmd `json:"hooks"`
	}
	hookURL := strings.TrimRight(baseURL, "/") + "/_hook"

	hooks := map[string][]hookGroup{}
	for _, ev := range events {
		url := fmt.Sprintf("%s?runtime=claude_code&event=%s&session=%s", hookURL, ev, sessionID)
		cmd := fmt.Sprintf("curl -sS -m 2 -X POST %q --data-binary @- >/dev/null 2>&1", url)
		hooks[ev] = []hookGroup{{Hooks: []hookCmd{{Type: "command", Command: cmd}}}}
	}
	b, _ := json.Marshal(map[string]any{"hooks": hooks})
	return string(b)
}
