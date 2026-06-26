package server

import (
	"strings"

	"github.com/jzlxiaohei/agenttape/internal/normalize"
	"github.com/jzlxiaohei/agenttape/internal/store"
)

// Compaction detection is graded by evidence strength, never by keyword guessing
// (see colleague review / REPLAY_LIB §7):
//
//	confirmed        — a PreCompact/PostCompact hook sits between the two requests
//	                   (harness ground truth).
//	strong_suspected — history shrank AND the prior response is carried into the
//	                   next request (content lineage: data provably flowed A→B).
//	weak_suspected   — history shrank only (could also be scroll-truncation, a
//	                   session reset, or a client rewrite — can't tell them apart).
//
// It is inherently cross-event: the judgment can only complete once request B has
// arrived, so we model a "compaction episode" over the (A, B) pair rather than
// tagging A at ingest.
type CompactionGrade string

const (
	gradeConfirmed CompactionGrade = "confirmed"
	gradeStrong    CompactionGrade = "strong_suspected"
	gradeWeak      CompactionGrade = "weak_suspected"
)

type CompactionEpisode struct {
	Grade       CompactionGrade `json:"grade"`
	BeforeEvent string          `json:"before_event"` // A — the request whose response is the summary
	AfterEvent  string          `json:"after_event"`  // B — the first request after the boundary
	Evidence    string          `json:"evidence"`
	ContextIn   int64           `json:"context_in"`  // A's full input incl. cache
	SummaryOut  int64           `json:"summary_out"` // A's output tokens (the summary)
}

// detectCompactionEpisodes pairs adjacent completions and grades each boundary.
func detectCompactionEpisodes(comps []store.CompletionRec, hooks []store.CompactHookRec) []CompactionEpisode {
	var out []CompactionEpisode
	for i := 0; i+1 < len(comps); i++ {
		a, b := comps[i], comps[i+1]
		if a.Env == nil || b.Env == nil || a.Env.Request == nil || b.Env.Request == nil {
			continue
		}
		shrank := historyShrank(a.Env.Request.Messages, b.Env.Request.Messages)
		hook := compactHookBetween(hooks, a.StartedAt, b.StartedAt)
		if !shrank && !hook {
			continue
		}
		carried := a.Env.Response != nil && responseCarried(a.Env.Response.FinalText, b.Env.Request.Messages)
		grade, evidence := gradeCompaction(hook, shrank, carried)
		if grade == "" {
			continue
		}
		ci, so := compactionTokens(a.Env)
		out = append(out, CompactionEpisode{
			Grade: grade, BeforeEvent: a.EventID, AfterEvent: b.EventID,
			Evidence: evidence, ContextIn: ci, SummaryOut: so,
		})
	}
	return out
}

func gradeCompaction(hook, shrank, carried bool) (CompactionGrade, string) {
	switch {
	case hook:
		return gradeConfirmed, "PreCompact/PostCompact hook between the two requests"
	case shrank && carried:
		return gradeStrong, "history shrank and the prior response is carried into the next request"
	case shrank:
		return gradeWeak, "history shrank, but no content lineage to confirm it (could be truncation/reset/rewrite)"
	}
	return "", ""
}

// historyShrank is true when B's message history collapsed relative to A's:
// a real compaction drops many turns and keeps a small summary, so we require a
// meaningful absolute drop AND at least a halving, on a non-trivial history.
const minHistoryForShrink = 6

func historyShrank(a, b []normalize.Message) bool {
	if len(a) < minHistoryForShrink {
		return false
	}
	removed := len(a) - len(b)
	return removed >= 4 && len(b)*2 <= len(a)
}

// responseCarried proves A's output flowed into B's input: it samples anchor
// chunks from A's response (the summary) and checks they appear verbatim in B's
// request text. This is content lineage, not a keyword guess — and it's what
// separates compaction (B carries A's summary) from truncation (B carries the
// tail of the old history, not A's output).
func responseCarried(summary string, b []normalize.Message) bool {
	s := normalizeWS(summary)
	if len(s) < 80 {
		return false
	}
	haystack := normalizeWS(messagesText(b))
	anchors := anchorChunks(s, 80, 5)
	if len(anchors) == 0 {
		return false
	}
	hits := 0
	for _, a := range anchors {
		if strings.Contains(haystack, a) {
			hits++
		}
	}
	return hits*2 >= len(anchors) // majority of anchors present
}

func compactHookBetween(hooks []store.CompactHookRec, aTime, bTime string) bool {
	for _, h := range hooks {
		// RFC3339Nano timestamps sort lexicographically (same format/zone), which is
		// the ordering the rest of the app already relies on.
		if h.StartedAt > aTime && h.StartedAt <= bTime {
			return true
		}
	}
	return false
}

func compactionTokens(env *normalize.NormalizedEnvelope) (int64, int64) {
	if env.Response == nil || env.Response.Usage == nil {
		return 0, 0
	}
	u := env.Response.Usage
	in := u.InputTokens + extraInt(u.Extra, "cache_read_input_tokens") + extraInt(u.Extra, "cache_creation_input_tokens")
	return in, u.OutputTokens
}

func extraInt(m map[string]any, k string) int64 {
	if m == nil {
		return 0
	}
	switch v := m[k].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	}
	return 0
}

// messagesText concatenates every text block across a message list.
func messagesText(msgs []normalize.Message) string {
	var sb strings.Builder
	for _, m := range msgs {
		for _, blk := range m.Content {
			if blk.Type == normalize.BlockText && blk.Text != "" {
				sb.WriteString(blk.Text)
				sb.WriteByte('\n')
			}
		}
	}
	return sb.String()
}

// normalizeWS collapses all whitespace runs to a single space so containment
// checks survive reformatting/re-wrapping at the edges.
func normalizeWS(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// anchorChunks samples up to n chunks of chunkLen runes, evenly spaced across s.
func anchorChunks(s string, chunkLen, n int) []string {
	r := []rune(s)
	if len(r) < chunkLen {
		return nil
	}
	maxStart := len(r) - chunkLen
	var out []string
	for i := 0; i < n; i++ {
		start := 0
		if n > 1 {
			start = maxStart * i / (n - 1)
		}
		out = append(out, string(r[start:start+chunkLen]))
	}
	return out
}
