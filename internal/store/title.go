package store

import (
	"regexp"
	"strings"

	"tracelab/internal/normalize"
)

var (
	reSystemReminder = regexp.MustCompile(`(?s)<system-reminder>.*?</system-reminder>`)
	reWhitespace     = regexp.MustCompile(`\s+`)
)

const maxTitleRunes = 100

// sessionTitle derives a friendly session title from the first user prompt in a
// completion request. cc/codex keep the original first user message at the head of
// every later request, so any completion yields the same title — we set it once.
// system-reminder wrappers (which cc/codex inject) are stripped so the title is the
// human's actual words, not harness boilerplate.
func sessionTitle(env *normalize.NormalizedEnvelope) string {
	if env == nil || env.Request == nil {
		return ""
	}
	for _, m := range env.Request.Messages {
		if m.Role != "user" {
			continue
		}
		var b strings.Builder
		for _, blk := range m.Content {
			if blk.Type == normalize.BlockText && blk.Text != "" {
				b.WriteString(blk.Text)
				b.WriteByte(' ')
			}
		}
		text := reSystemReminder.ReplaceAllString(b.String(), " ")
		text = strings.TrimSpace(reWhitespace.ReplaceAllString(text, " "))
		if text == "" {
			continue // e.g. a turn that was only a system-reminder
		}
		return truncateRunes(text, maxTitleRunes)
	}
	return ""
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimSpace(string(r[:n])) + "…"
}
