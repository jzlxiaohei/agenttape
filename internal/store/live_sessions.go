package store

import "time"

// LiveSessionRow is the persisted, NON-SECRET routing of a proxy session — enough
// to re-attach a still-running agent after a agenttape restart, and nothing more.
// No credentials: token is a routing handle, and a key-mode session's real key is
// never stored (it lives only in process memory). See docs/SECURITY.md.
type LiveSessionRow struct {
	ID       string
	Token    string
	Client   string
	Upstream string
	Provider string
	Mode     string
}

// SaveLiveSession upserts a live-session routing record.
func (s *Store) SaveLiveSession(r LiveSessionRow) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO live_sessions(id, token, client, upstream, provider, mode, created_at)
		 VALUES(?,?,?,?,?,?,?)`,
		r.ID, r.Token, r.Client, r.Upstream, r.Provider, r.Mode,
		time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

// DeleteLiveSession forgets a live-session routing record (e.g. when the user
// closes the session).
func (s *Store) DeleteLiveSession(id string) error {
	_, err := s.db.Exec(`DELETE FROM live_sessions WHERE id = ?`, id)
	return err
}

// AllLiveSessions returns every persisted live-session routing record, used to
// rehydrate the in-memory registry at startup.
func (s *Store) AllLiveSessions() ([]LiveSessionRow, error) {
	rows, err := s.db.Query(
		`SELECT id, token, client, upstream, provider, mode FROM live_sessions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LiveSessionRow{}
	for rows.Next() {
		var r LiveSessionRow
		if err := rows.Scan(&r.ID, &r.Token, &r.Client, &r.Upstream, &r.Provider, &r.Mode); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
