package store

import (
	"strings"
	"testing"

	"tracelab/internal/normalize"
)

func userMsg(blocks ...string) normalize.Message {
	m := normalize.Message{Role: "user"}
	for _, b := range blocks {
		m.Content = append(m.Content, normalize.ContentBlock{Type: normalize.BlockText, Text: b})
	}
	return m
}

func TestSessionTitle(t *testing.T) {
	cases := []struct {
		name string
		env  *normalize.NormalizedEnvelope
		want string
	}{
		{"nil", nil, ""},
		{
			"strips system-reminder, keeps real prompt",
			&normalize.NormalizedEnvelope{Request: &normalize.RequestEnvelope{Messages: []normalize.Message{
				userMsg("<system-reminder>\nbe nice\n</system-reminder>\n\n", "你是谁"),
			}}},
			"你是谁",
		},
		{
			"skips leading assistant/tool, uses first user",
			&normalize.NormalizedEnvelope{Request: &normalize.RequestEnvelope{Messages: []normalize.Message{
				{Role: "assistant", Content: []normalize.ContentBlock{{Type: normalize.BlockText, Text: "hi"}}},
				userMsg("fix the build"),
			}}},
			"fix the build",
		},
		{
			"collapses whitespace",
			&normalize.NormalizedEnvelope{Request: &normalize.RequestEnvelope{Messages: []normalize.Message{
				userMsg("line one\n\n   line two"),
			}}},
			"line one line two",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := sessionTitle(c.env); got != c.want {
				t.Errorf("sessionTitle = %q, want %q", got, c.want)
			}
		})
	}
}

func TestSessionTitleTruncates(t *testing.T) {
	long := strings.Repeat("好", 250)
	env := &normalize.NormalizedEnvelope{Request: &normalize.RequestEnvelope{Messages: []normalize.Message{userMsg(long)}}}
	got := sessionTitle(env)
	if r := []rune(got); len(r) != maxTitleRunes+1 || !strings.HasSuffix(got, "…") {
		t.Errorf("expected %d runes + ellipsis, got %d runes: %q", maxTitleRunes, len(r), got)
	}
}
