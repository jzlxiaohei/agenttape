package launcher

import (
	"fmt"
	"regexp"
	"strings"

	"tracelab/internal/source/hook"
	"tracelab/internal/source/httpcap"
)

// The Codex DESKTOP app cannot take per-invocation `-c` overrides (its background
// process doesn't inherit them), so the only way to route it through the proxy is
// writing ~/.codex/config.toml. That breaks the usual non-invasive rule (next.md
// 1.1), so the server pairs every write with a verbatim backup + restore. These
// markers make the injection recognizable for conflict detection and manual
// cleanup; restore itself is byte-exact from the backup, not marker-based.
const (
	CodexMarker    = "# tracelab: Codex desktop capture (auto-injected — restore via the tracelab launch page)"
	codexBlockOpen = "# >>> tracelab capture (auto-injected) >>>"
	codexBlockEnd  = "# <<< tracelab capture <<<"
)

var topLevelModelProvider = regexp.MustCompile(`(?m)^\s*model_provider\s*=`)

// MergeCodexDesktopConfig returns config.toml content that routes the Codex desktop
// app through the proxy (subscription auth) and, optionally, injects hook capture —
// while preserving the user's other settings.
//
// TOML ordering forces the shape: top-level keys must precede any table, tables
// must follow. So we drop the original's top-level `model_provider` (we own it),
// prepend ours, keep the user's body, then append our provider + hook tables. The
// active file is reformatted only cosmetically; the byte-exact original is held in
// the backup, so restore is lossless regardless.
func MergeCodexDesktopConfig(original, serverURL string, sess *httpcap.Session, events []string) string {
	withHooks := len(events) > 0
	base := httpcap.SessionBaseURL(serverURL, sess)
	stripped := strings.TrimRight(stripTopLevelModelProvider(original), "\n")

	var b strings.Builder
	b.WriteString(CodexMarker + "\n")
	b.WriteString(`model_provider = "tracelab"` + "\n")
	if stripped != "" {
		b.WriteString("\n" + stripped + "\n")
	}
	b.WriteString("\n" + codexBlockOpen + "\n")
	b.WriteString("[model_providers.tracelab]\n")
	b.WriteString(`name = "tracelab capture"` + "\n")
	fmt.Fprintf(&b, "base_url = %q\n", base)
	b.WriteString(`wire_api = "responses"` + "\n")
	b.WriteString("requires_openai_auth = true\n")
	if withHooks {
		b.WriteString("\n" + hook.CodexHooksTOML(events, serverURL, sess.ID))
	}
	b.WriteString(codexBlockEnd + "\n")
	return b.String()
}

// stripTopLevelModelProvider removes a `model_provider = …` assignment only when it
// is a real top-level key (before the first table header), leaving profile/table-
// scoped ones (e.g. [profiles.x] model_provider=…) untouched.
func stripTopLevelModelProvider(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	atTop := true
	for _, ln := range lines {
		if atTop && strings.HasPrefix(strings.TrimSpace(ln), "[") {
			atTop = false
		}
		if atTop && topLevelModelProvider.MatchString(ln) {
			continue
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
}

// HasCodexMarker reports whether config already carries tracelab's injection.
func HasCodexMarker(content string) bool {
	return strings.Contains(content, CodexMarker)
}
