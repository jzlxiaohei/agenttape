package store

import (
	"database/sql"
	"strings"
	"time"

	"github.com/jzlxiaohei/agenttape/internal/event"
	"github.com/jzlxiaohei/agenttape/internal/sink"
)

// tagRow is one tag to persist.
type tagRow struct {
	tag        string
	confidence float64
	suspected  bool
	source     string // structural | heuristic | manual
	evidence   string
}

// insertTags persists structural signals from normalization plus accurate
// hook-derived tags. Accuracy is the priority (next.md decision 3): we only emit
// a confident tag when the structure or the hook name proves it; anything
// uncertain stays suspected with evidence.
func insertTags(tx *sql.Tx, rec sink.Record) error {
	var rows []tagRow

	if rec.Normalized != nil {
		for _, sig := range rec.Normalized.Signals {
			src := "structural"
			if sig.Suspected {
				src = "heuristic"
			}
			rows = append(rows, tagRow{sig.Tag, sig.Confidence, sig.Suspected, src, sig.Evidence})
		}
	}
	rows = append(rows, hookTags(rec.Event.Hook)...)

	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, r := range rows {
		if _, err := tx.Exec(
			`INSERT INTO tags(event_id, tag, confidence, suspected, source, evidence, created_at)
			 VALUES(?,?,?,?,?,?,?)`,
			rec.Event.ID, r.tag, r.confidence, boolInt(r.suspected), r.source, r.evidence, now); err != nil {
			return err
		}
	}
	return nil
}

// hookTags derives tags from a hook event's name. These are facts (the harness
// itself names the event), so they are structural, not guesses. Subagent
// detection via hooks is more reliable than inferring it from HTTP traffic.
func hookTags(h *event.HookEvent) []tagRow {
	if h == nil {
		return nil
	}
	name := strings.ToLower(h.EventName)
	var rows []tagRow
	if strings.Contains(name, "subagent") {
		rows = append(rows, tagRow{"subagent", 1, false, "structural", "hook event: " + h.EventName})
	}
	if strings.Contains(name, "tooluse") || strings.Contains(name, "tool_use") {
		rows = append(rows, tagRow{"tool_call", 1, false, "structural", "hook event: " + h.EventName})
	}
	return rows
}
