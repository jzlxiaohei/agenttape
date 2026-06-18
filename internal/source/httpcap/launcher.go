package httpcap

import (
	"os"
	"os/exec"
	"strings"
)

// SessionBaseURL is the per-session proxy entrypoint a client should be pointed
// at: <proxyBase>/s/<token>.
func SessionBaseURL(proxyBase string, sess *Session) string {
	return strings.TrimRight(proxyBase, "/") + "/s/" + sess.Token
}

// LaunchClaudeCode builds a command that runs Claude Code through the proxy by
// setting ANTHROPIC_BASE_URL for this process only. It writes NO global config
// and mutates NO files — the parent environment is left untouched (next.md 1.1).
func LaunchClaudeCode(proxyBase string, sess *Session, args ...string) *exec.Cmd {
	cmd := exec.Command("claude", args...)
	cmd.Env = append(os.Environ(), "ANTHROPIC_BASE_URL="+SessionBaseURL(proxyBase, sess))
	return cmd
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
