package server

import (
	"strings"
	"testing"

	"tracelab/internal/normalize"
	"tracelab/internal/store"
)

func TestGradeCompaction(t *testing.T) {
	cases := []struct {
		hook, shrank, carried bool
		want                  CompactionGrade
	}{
		{true, false, false, gradeConfirmed},  // hook wins regardless
		{true, true, true, gradeConfirmed},     //
		{false, true, true, gradeStrong},       // lineage proven
		{false, true, false, gradeWeak},        // shrink only
		{false, false, true, ""},               // carried but no shrink → normal turn, not compaction
		{false, false, false, ""},              //
	}
	for _, c := range cases {
		got, _ := gradeCompaction(c.hook, c.shrank, c.carried)
		if got != c.want {
			t.Errorf("gradeCompaction(hook=%v shrank=%v carried=%v) = %q, want %q", c.hook, c.shrank, c.carried, got, c.want)
		}
	}
}

func msgs(n int) []normalize.Message {
	out := make([]normalize.Message, n)
	for i := range out {
		out[i] = normalize.Message{Role: "user", Content: []normalize.ContentBlock{{Type: normalize.BlockText, Text: "m"}}}
	}
	return out
}

func TestHistoryShrank(t *testing.T) {
	if !historyShrank(msgs(40), msgs(3)) {
		t.Error("40→3 should shrink")
	}
	if historyShrank(msgs(40), msgs(38)) {
		t.Error("40→38 (only 2 dropped) should not shrink")
	}
	if historyShrank(msgs(40), msgs(25)) {
		t.Error("40→25 (not halved) should not shrink")
	}
	if historyShrank(msgs(4), msgs(1)) {
		t.Error("tiny history (4) should not qualify")
	}
	if historyShrank(msgs(40), msgs(41)) {
		t.Error("history that grew should not shrink")
	}
}

func textMsg(s string) normalize.Message {
	return normalize.Message{Role: "user", Content: []normalize.ContentBlock{{Type: normalize.BlockText, Text: s}}}
}

func TestResponseCarried(t *testing.T) {
	summary := strings.Repeat("The project is a CLI log tool. ", 20) // long, distinctive
	// B carries the summary near-verbatim (compaction continuation) → carried.
	b := []normalize.Message{textMsg("This session is being continued from a previous conversation.\n" + summary)}
	if !responseCarried(summary, b) {
		t.Error("summary embedded in B should be carried")
	}
	// B carries unrelated tail text (truncation look-alike) → not carried.
	tail := []normalize.Message{textMsg(strings.Repeat("unrelated tail content here. ", 20))}
	if responseCarried(summary, tail) {
		t.Error("unrelated content should not count as carried")
	}
	// Short/empty summary → not carried.
	if responseCarried("tiny", b) {
		t.Error("short summary should not be carried")
	}
}

func TestCompactHookBetween(t *testing.T) {
	hooks := []store.CompactHookRec{{EventName: "PreCompact", StartedAt: "2026-06-23T10:00:05Z"}}
	if !compactHookBetween(hooks, "2026-06-23T10:00:00Z", "2026-06-23T10:00:10Z") {
		t.Error("hook inside the interval should match")
	}
	if compactHookBetween(hooks, "2026-06-23T10:00:06Z", "2026-06-23T10:00:10Z") {
		t.Error("hook before the interval should not match")
	}
}

func TestDetectEpisodes_StrongAndConfirmed(t *testing.T) {
	summary := strings.Repeat("Summary line about the work done so far. ", 20)
	a := store.CompletionRec{
		EventID: "A", StartedAt: "2026-06-23T10:00:00Z",
		Env: &normalize.NormalizedEnvelope{
			Request:  &normalize.RequestEnvelope{Messages: msgs(40)},
			Response: &normalize.ResponseEnvelope{FinalText: summary, Usage: &normalize.Usage{InputTokens: 1900, OutputTokens: 5000, Extra: map[string]any{"cache_read_input_tokens": float64(96000)}}},
		},
	}
	b := store.CompletionRec{
		EventID: "B", StartedAt: "2026-06-23T10:00:10Z",
		Env: &normalize.NormalizedEnvelope{
			Request: &normalize.RequestEnvelope{Messages: []normalize.Message{textMsg(summary)}},
		},
	}
	// No hooks → strong (shrank + carried), tokens come from A incl. cache.
	eps := detectCompactionEpisodes([]store.CompletionRec{a, b}, nil)
	if len(eps) != 1 || eps[0].Grade != gradeStrong {
		t.Fatalf("want one strong episode, got %+v", eps)
	}
	if eps[0].ContextIn != 97900 || eps[0].SummaryOut != 5000 {
		t.Errorf("tokens = in %d out %d, want 97900/5000", eps[0].ContextIn, eps[0].SummaryOut)
	}
	// With a PreCompact hook between → confirmed.
	hooks := []store.CompactHookRec{{EventName: "PreCompact", StartedAt: "2026-06-23T10:00:05Z"}}
	eps = detectCompactionEpisodes([]store.CompletionRec{a, b}, hooks)
	if len(eps) != 1 || eps[0].Grade != gradeConfirmed {
		t.Fatalf("want one confirmed episode, got %+v", eps)
	}
}
