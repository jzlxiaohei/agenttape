package hook

import (
	"encoding/json"
	"fmt"
	"strings"
)

// claudeEvents is the full set of Claude Code hook events we capture to study
// harness orchestration (next.md 7.1). Sourced from the official docs
// (code.claude.com/docs/en/hooks). We capture everything EXCEPT MessageDisplay,
// which fires continuously while assistant text streams and would flood capture.
// Unknown event names are harmlessly ignored by older Claude Code versions.
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

// BuildClaudeSettings returns a Claude Code settings JSON string wiring each
// lifecycle event to a curl that ships the hook payload to /_hook.
func BuildClaudeSettings(baseURL, sessionID string) string {
	type hookCmd struct {
		Type    string `json:"type"`
		Command string `json:"command"`
	}
	type hookGroup struct {
		Hooks []hookCmd `json:"hooks"`
	}
	hookURL := strings.TrimRight(baseURL, "/") + "/_hook"

	hooks := map[string][]hookGroup{}
	for _, ev := range claudeEvents {
		url := fmt.Sprintf("%s?runtime=claude_code&event=%s&session=%s", hookURL, ev, sessionID)
		cmd := fmt.Sprintf("curl -sS -m 2 -X POST %q --data-binary @- >/dev/null 2>&1", url)
		hooks[ev] = []hookGroup{{Hooks: []hookCmd{{Type: "command", Command: cmd}}}}
	}
	b, _ := json.Marshal(map[string]any{"hooks": hooks})
	return string(b)
}
