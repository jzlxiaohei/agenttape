package store

import "strings"

// SearchResult is one hit in cross-session search: enough to render a result row
// and navigate to the event.
type SearchResult struct {
	EventID   string `json:"event_id"`
	SessionID string `json:"session_id"`
	Client    string `json:"client"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	StartedAt string `json:"started_at"`
	Snippet   string `json:"snippet"`
}

// SearchFilters narrows results. Empty fields are ignored.
type SearchFilters struct {
	Query    string // FTS5 query; empty = browse recent (filtered)
	Tag      string
	Provider string
	Client   string
}

// SearchEvents runs full-text search (when Query is set) plus structured
// filters, newest-first. With no Query it browses recent events matching the
// filters.
func (s *Store) SearchEvents(f SearchFilters) ([]SearchResult, error) {
	var (
		joins  []string
		wheres []string
		args   []any
		sel    = "''"
	)
	if q := strings.TrimSpace(f.Query); q != "" {
		joins = append(joins, "JOIN events_fts fts ON fts.event_id = e.id")
		wheres = append(wheres, "events_fts MATCH ?")
		args = append(args, q)
		sel = "snippet(events_fts, -1, '[', ']', '…', 12)"
	}
	if f.Tag != "" {
		wheres = append(wheres, "EXISTS (SELECT 1 FROM tags t WHERE t.event_id = e.id AND t.tag = ?)")
		args = append(args, f.Tag)
	}
	if f.Provider != "" {
		wheres = append(wheres, "h.provider = ?")
		args = append(args, f.Provider)
	}
	if f.Client != "" {
		wheres = append(wheres, "s.client = ?")
		args = append(args, f.Client)
	}

	sql := `SELECT e.id, e.session_id, COALESCE(s.client,''), COALESCE(h.provider,''),
	          COALESCE(h.model,''), e.started_at, ` + sel + `
	        FROM events e
	          LEFT JOIN sessions s ON s.id = e.session_id
	          LEFT JOIN http_exchanges h ON h.event_id = e.id
	          ` + strings.Join(joins, " ")
	if len(wheres) > 0 {
		sql += " WHERE " + strings.Join(wheres, " AND ")
	}
	sql += " ORDER BY e.started_at DESC LIMIT 200"

	rows, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SearchResult{}
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.EventID, &r.SessionID, &r.Client, &r.Provider, &r.Model, &r.StartedAt, &r.Snippet); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Facets returns the distinct providers, clients, and tags present, for building
// filter dropdowns.
func (s *Store) Facets() (providers, clients, tags []string, err error) {
	providers, err = s.distinct("SELECT DISTINCT provider FROM http_exchanges WHERE provider <> '' ORDER BY provider")
	if err != nil {
		return
	}
	clients, err = s.distinct("SELECT DISTINCT client FROM sessions WHERE client <> '' ORDER BY client")
	if err != nil {
		return
	}
	tags, err = s.distinct("SELECT DISTINCT tag FROM tags ORDER BY tag")
	return
}

func (s *Store) distinct(query string) ([]string, error) {
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
