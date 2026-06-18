package store

// SessionSummary is a row for the session list (M3 will render these as tabs).
type SessionSummary struct {
	ID         string `json:"id"`
	Client     string `json:"client"`
	Upstream   string `json:"upstream"`
	StartedAt  string `json:"started_at"`
	EventCount int    `json:"event_count"`
}

// ListSessions returns sessions with their event counts, newest first.
func (s *Store) ListSessions() ([]SessionSummary, error) {
	rows, err := s.db.Query(
		`SELECT s.id, s.client, s.upstream, s.started_at, COUNT(e.id)
		 FROM sessions s LEFT JOIN events e ON e.session_id = s.id
		 GROUP BY s.id ORDER BY s.started_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SessionSummary
	for rows.Next() {
		var ss SessionSummary
		if err := rows.Scan(&ss.ID, &ss.Client, &ss.Upstream, &ss.StartedAt, &ss.EventCount); err != nil {
			return nil, err
		}
		out = append(out, ss)
	}
	return out, rows.Err()
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
