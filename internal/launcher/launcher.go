// Package launcher builds client process invocations that connect a coding
// agent to agenttape's capture adapters. It is the composition layer: HTTP
// traffic is pointed at httpcap, while harness lifecycle/tool events are wired
// to hook capture.
package launcher

import (
	"os"
	"os/exec"

	"github.com/jzlxiaohei/agenttape/internal/source/hook"
	"github.com/jzlxiaohei/agenttape/internal/source/httpcap"
)

// LaunchClaudeCode builds a command that runs Claude Code through the proxy
// (ANTHROPIC_BASE_URL) and injects hook capture via `--settings` with a JSON
// string. Both apply to this invocation only — NO global config or files are
// mutated (next.md 1.1). The hooks POST each event to the server's /_hook
// endpoint stamped with this session's id, so hook events correlate with the
// HTTP captures of the same session.
func LaunchClaudeCode(serverURL string, sess *httpcap.Session, events []string, args ...string) *exec.Cmd {
	settings := hook.BuildClaudeSettings(events, serverURL, sess.ID)
	full := append([]string{"--settings", settings}, args...)
	cmd := exec.Command("claude", full...)
	cmd.Env = append(os.Environ(),
		"ANTHROPIC_BASE_URL="+httpcap.SessionBaseURL(serverURL, sess),
		"AGENTTAPE_SESSION="+sess.ID,
	)
	return cmd
}

// LaunchCodex builds a command that runs the Codex CLI through the proxy using
// single-run `-c` overrides. This is the clean, non-invasive alternative to
// editing ~/.codex/config.toml: the overrides apply to this invocation only and
// nothing on disk is changed (next.md 1.1). The same `-c` mechanism injects the
// hook capture table, so hook events correlate with the session's HTTP captures
// exactly like Claude Code.
//
// Codex records trust against each command hook's hash and skips untrusted
// hooks. Because we inject hooks per-invocation (never persisting a trust hash),
// we pass --dangerously-bypass-hook-trust so OUR vetted hooks run for this one
// run only — no global trust store or config file is touched, keeping the launch
// non-invasive.
//
// Codex Desktop App is intentionally NOT handled here — its background process
// does not reliably inherit `-c` overrides, and we refuse to silently rewrite
// the user's global config as a workaround. That limitation is documented rather
// than worked around.
func LaunchCodex(serverURL string, sess *httpcap.Session, events []string, args ...string) *exec.Cmd {
	base := httpcap.SessionBaseURL(serverURL, sess)
	overrides := []string{
		"-c", `model_provider="agenttape"`,
		"-c", `model_providers.agenttape.name="agenttape capture"`,
		"-c", `model_providers.agenttape.base_url="` + base + `"`,
		"-c", `model_providers.agenttape.wire_api="responses"`,
		"-c", `model_providers.agenttape.requires_openai_auth=true`,
	}
	overrides = append(overrides, hook.BuildCodexOverrides(events, serverURL, sess.ID)...)
	overrides = append(overrides, "--dangerously-bypass-hook-trust")
	cmd := exec.Command("codex", append(overrides, args...)...)
	cmd.Env = os.Environ()
	return cmd
}
