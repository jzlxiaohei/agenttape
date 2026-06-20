package hook

import (
	"fmt"
	"strings"
)

// codexEvents is the full set of Codex CLI hook events (codex >= 0.141).
// Sourced from the official docs (developers.openai.com/codex/hooks). Each
// fires a command that receives a JSON payload on stdin — the SAME handoff shape
// as Claude Code's hooks, so the /_hook endpoint treats both runtimes
// identically.
var codexEvents = []string{
	// session & turn lifecycle
	"SessionStart", "UserPromptSubmit", "Stop",
	// agentic loop / tools
	"PreToolUse", "PostToolUse", "PermissionRequest",
	// subagents
	"SubagentStart", "SubagentStop",
	// compaction
	"PreCompact", "PostCompact",
}

// BuildCodexOverrides returns one `-c hooks.<Event>=[...]` override per hook
// event, wiring each to a curl that ships the stdin payload to /_hook stamped
// with this session's id. The value is inline TOML (codex parses `-c` as TOML):
// a single matcher group whose only handler is our command. An empty matcher
// matches everything.
func BuildCodexOverrides(baseURL, sessionID string) []string {
	hookURL := strings.TrimRight(baseURL, "/") + "/_hook"
	out := make([]string, 0, len(codexEvents)*2)
	for _, ev := range codexEvents {
		url := fmt.Sprintf("%s?runtime=codex&event=%s&session=%s", hookURL, ev, sessionID)
		curl := fmt.Sprintf(`curl -sS -m 2 -X POST %q --data-binary @- >/dev/null 2>&1`, url)
		val := fmt.Sprintf(`hooks.%s=[{matcher="", hooks=[{type="command", command=%s}]}]`,
			ev, tomlString(curl))
		out = append(out, "-c", val)
	}
	return out
}

// tomlString renders s as a TOML basic (double-quoted) string, escaping the
// characters TOML requires so an arbitrary shell command survives as a `-c`
// value.
func tomlString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
