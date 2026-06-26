package hook

import (
	"fmt"
	"strings"
)

// codexEvents is the BUILT-IN DEFAULT set of Codex CLI hook events (codex >=
// 0.141), used to seed the store's per-client registry on first run. It is not
// read at launch — the launcher reads the user-editable enabled set from the
// store — so a user can add a newly-shipped Codex event without a agenttape
// release. Sourced from the official docs (developers.openai.com/codex/hooks).
// Each fires a command that receives a JSON payload on stdin — the SAME handoff
// shape as Claude Code's hooks, so the /_hook endpoint treats both runtimes
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

// codexHookCommand is the curl that ships a hook event's stdin payload to /_hook
// stamped with runtime/event/session, shared by the -c (CLI) and config.toml
// (desktop) injection paths so both fire identical capture.
func codexHookCommand(baseURL, sessionID, event string) string {
	hookURL := strings.TrimRight(baseURL, "/") + "/_hook"
	url := fmt.Sprintf("%s?runtime=codex&event=%s&session=%s", hookURL, event, sessionID)
	return fmt.Sprintf(`curl -sS -m 2 -X POST %q --data-binary @- >/dev/null 2>&1`, url)
}

// DefaultCodexEvents returns a copy of the built-in default Codex CLI event set,
// used to seed the store and as the launcher's fallback when the store is
// unreachable.
func DefaultCodexEvents() []string {
	return append([]string(nil), codexEvents...)
}

// BuildCodexOverrides returns one `-c hooks.<Event>=[...]` override per given
// event, wiring each to a curl that ships the stdin payload to /_hook stamped
// with this session's id. The value is inline TOML (codex parses `-c` as TOML):
// a single matcher group whose only handler is our command. An empty matcher
// matches everything.
func BuildCodexOverrides(events []string, baseURL, sessionID string) []string {
	out := make([]string, 0, len(events)*2)
	for _, ev := range events {
		val := fmt.Sprintf(`hooks.%s=[{matcher="", hooks=[{type="command", command=%s}]}]`,
			ev, tomlString(codexHookCommand(baseURL, sessionID, ev)))
		out = append(out, "-c", val)
	}
	return out
}

// CodexHooksTOML renders the same capture as BuildCodexOverrides but as persistent
// config.toml array-of-tables, for the Codex DESKTOP app — which can't take
// per-invocation `-c` overrides. Note: codex still gates command hooks behind
// hash-based trust, and the desktop app has no --dangerously-bypass-hook-trust;
// the user trusts these once via the app's /hooks before they fire.
func CodexHooksTOML(events []string, baseURL, sessionID string) string {
	var b strings.Builder
	for _, ev := range events {
		fmt.Fprintf(&b, "[[hooks.%s]]\n", ev)
		b.WriteString("matcher = \"\"\n")
		fmt.Fprintf(&b, "[[hooks.%s.hooks]]\n", ev)
		b.WriteString("type = \"command\"\n")
		fmt.Fprintf(&b, "command = %s\n", tomlString(codexHookCommand(baseURL, sessionID, ev)))
	}
	return b.String()
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
