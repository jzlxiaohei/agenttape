package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// SessionSummary is a row for the session list (M3 will render these as tabs).
type SessionSummary struct {
	ID         string `json:"id"`
	Client     string `json:"client"`
	Upstream   string `json:"upstream"`
	Title      string `json:"title"` // derived from the first user prompt (may be empty)
	StartedAt  string `json:"started_at"`
	EventCount int    `json:"event_count"`
	// HookCount is how many captured events are harness hooks. Zero on a session
	// that has traffic means it was launched HTTP-only (the env "run yourself" path,
	// no hook injection) — the UI badges these.
	HookCount int `json:"hook_count"`
}

const hookCountExpr = `COALESCE(SUM(CASE WHEN e.kind = 'hook_event' THEN 1 ELSE 0 END), 0)`

// ListSessions returns sessions with their event/hook counts, newest first.
func (s *Store) ListSessions() ([]SessionSummary, error) {
	rows, err := s.db.Query(
		`SELECT s.id, s.client, s.upstream, COALESCE(s.label,''), s.started_at, COUNT(e.id), ` + hookCountExpr + `
		 FROM sessions s LEFT JOIN events e ON e.session_id = s.id
		 GROUP BY s.id ORDER BY s.started_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SessionSummary
	for rows.Next() {
		var ss SessionSummary
		if err := rows.Scan(&ss.ID, &ss.Client, &ss.Upstream, &ss.Title, &ss.StartedAt, &ss.EventCount, &ss.HookCount); err != nil {
			return nil, err
		}
		out = append(out, ss)
	}
	return out, rows.Err()
}

// GetSession returns one persisted session summary by id.
func (s *Store) GetSession(id string) (*SessionSummary, error) {
	var ss SessionSummary
	err := s.db.QueryRow(
		`SELECT s.id, s.client, s.upstream, COALESCE(s.label,''), s.started_at, COUNT(e.id), `+hookCountExpr+`
		 FROM sessions s LEFT JOIN events e ON e.session_id = s.id
		 WHERE s.id = ?
		 GROUP BY s.id`, id).Scan(
		&ss.ID, &ss.Client, &ss.Upstream, &ss.Title, &ss.StartedAt, &ss.EventCount, &ss.HookCount)
	if err != nil {
		return nil, err
	}
	return &ss, nil
}

// SetSessionLabel sets a user-chosen name for a session, overriding the title that
// was auto-derived from the first prompt. An empty label clears it (the auto-fill
// in http.go may re-derive one from a later completion). Idempotent.
func (s *Store) SetSessionLabel(id, label string) error {
	_, err := s.db.Exec(`UPDATE sessions SET label = ? WHERE id = ?`, label, id)
	return err
}

// DeleteSession permanently removes a session and ALL its captured data: every
// event plus the per-type detail rows, tags, sections, FTS entries, raw-file
// pointers, and the raw bytes on disk. Foreign keys are enforced, so children are
// cleared before their events. Irreversible — gated behind a confirm in the UI.
func (s *Store) DeleteSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Child tables keyed by event_id — clear them for every event in this session.
	for _, tbl := range []string{"http_exchanges", "hook_events", "raw_files", "tags", "sections", "events_fts"} {
		if _, err := tx.Exec(
			`DELETE FROM `+tbl+` WHERE event_id IN (SELECT id FROM events WHERE session_id = ?)`, id); err != nil {
			return fmt.Errorf("delete %s: %w", tbl, err)
		}
	}
	if _, err := tx.Exec(`DELETE FROM events WHERE session_id = ?`, id); err != nil {
		return fmt.Errorf("delete events: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM sessions WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	// Best-effort: drop the on-disk raw bytes for this session (a whole directory
	// under raw/<session>/). A leftover dir is harmless, so a failure here doesn't
	// fail the delete.
	_ = os.RemoveAll(filepath.Join(s.rawDir, id))
	return nil
}

// Search returns event ids whose prompt text matches an FTS5 query.
func (s *Store) Search(query string) ([]string, error) {
	rows, err := s.db.Query(
		`SELECT event_id FROM events_fts WHERE events_fts MATCH ? ORDER BY rank`, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// TagCounts returns how many events carry each tag.
func (s *Store) TagCounts() (map[string]int, error) {
	rows, err := s.db.Query(`SELECT tag, COUNT(*) FROM tags GROUP BY tag ORDER BY COUNT(*) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var tag string
		var n int
		if err := rows.Scan(&tag, &n); err != nil {
			return nil, err
		}
		out[tag] = n
	}
	return out, rows.Err()
}
