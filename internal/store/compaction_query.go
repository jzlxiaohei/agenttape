package store

import (
	"encoding/json"

	"tracelab/internal/normalize"
)

// CompletionRec is one captured completion of a session, with its normalized
// envelope parsed — the input the compaction-episode detector pairs up.
type CompletionRec struct {
	EventID   string
	StartedAt string
	Env       *normalize.NormalizedEnvelope
}

// CompactHookRec is a PreCompact/PostCompact hook occurrence (harness ground
// truth for compaction).
type CompactHookRec struct {
	EventName string
	StartedAt string
}

// CompactionInputs returns a session's completions (chronological, with parsed
// normalized envelopes) and its PreCompact/PostCompact hooks. Compaction is a
// cross-request judgment, so the detector needs the whole session at once — it
// can only be decided once the request AFTER a candidate boundary has arrived.
func (s *Store) CompactionInputs(sessionID string) ([]CompletionRec, []CompactHookRec, error) {
	rows, err := s.db.Query(
		`SELECT e.id, e.started_at, COALESCE(h.normalized_json,'')
		 FROM events e JOIN http_exchanges h ON h.event_id = e.id
		 WHERE e.session_id = ? AND h.is_completion = 1
		 ORDER BY e.started_at, e.created_at`, sessionID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var comps []CompletionRec
	for rows.Next() {
		var c CompletionRec
		var nj string
		if err := rows.Scan(&c.EventID, &c.StartedAt, &nj); err != nil {
			return nil, nil, err
		}
		if nj != "" {
			var env normalize.NormalizedEnvelope
			if json.Unmarshal([]byte(nj), &env) == nil {
				c.Env = &env
			}
		}
		comps = append(comps, c)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	hrows, err := s.db.Query(
		`SELECT hk.event_name, e.started_at
		 FROM events e JOIN hook_events hk ON hk.event_id = e.id
		 WHERE e.session_id = ? AND hk.event_name IN ('PreCompact','PostCompact')
		 ORDER BY e.started_at`, sessionID)
	if err != nil {
		return nil, nil, err
	}
	defer hrows.Close()
	var hooks []CompactHookRec
	for hrows.Next() {
		var h CompactHookRec
		if err := hrows.Scan(&h.EventName, &h.StartedAt); err != nil {
			return nil, nil, err
		}
		hooks = append(hooks, h)
	}
	return comps, hooks, hrows.Err()
}
