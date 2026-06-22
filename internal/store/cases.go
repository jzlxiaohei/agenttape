package store

import "time"

// ReplayCase is one saved, re-sendable request in the replay library.
type ReplayCase struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Tags      string `json:"tags"`
	Provider  string `json:"provider"`
	Method    string `json:"method"`
	Target    string `json:"target"`
	Body      string `json:"body"`
	Source    string `json:"source"` // seed | captured
	CreatedAt string `json:"created_at"`
}

// ListCases returns all replay-library cases, newest first.
func (s *Store) ListCases() ([]ReplayCase, error) {
	rows, err := s.db.Query(
		`SELECT id, name, tags, provider, method, target, body, source, created_at
		 FROM replay_cases ORDER BY created_at DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReplayCase{}
	for rows.Next() {
		var c ReplayCase
		if err := rows.Scan(&c.ID, &c.Name, &c.Tags, &c.Provider, &c.Method, &c.Target, &c.Body, &c.Source, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCase returns one case or sql.ErrNoRows.
func (s *Store) GetCase(id string) (*ReplayCase, error) {
	var c ReplayCase
	err := s.db.QueryRow(
		`SELECT id, name, tags, provider, method, target, body, source, created_at
		 FROM replay_cases WHERE id = ?`, id).Scan(
		&c.ID, &c.Name, &c.Tags, &c.Provider, &c.Method, &c.Target, &c.Body, &c.Source, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// AddCase inserts (or replaces) a case.
func (s *Store) AddCase(c ReplayCase) error {
	if c.CreatedAt == "" {
		c.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO replay_cases(id, name, tags, provider, method, target, body, source, created_at)
		 VALUES(?,?,?,?,?,?,?,?,?)`,
		c.ID, c.Name, c.Tags, c.Provider, c.Method, c.Target, c.Body, c.Source, c.CreatedAt)
	return err
}

// seedCases installs predefined cases once (fixed ids, INSERT OR IGNORE so user
// edits/deletes are not clobbered on restart). The simplest experiment: ask the
// model "who are you" — one per wire format.
func (s *Store) seedCases() error {
	seeds := []ReplayCase{
		{
			ID:       "seed:whoami-codex",
			Name:     "你是谁 (codex / responses)",
			Tags:     "text,smoke",
			Provider: "openai-responses",
			Method:   "POST",
			Target:   "https://api.openai.com/v1/responses",
			Body:     `{"model":"gpt-5.5","input":"你是谁?","stream":true}`,
			Source:   "seed",
		},
		{
			ID:       "seed:whoami-claude",
			Name:     "你是谁 (claude / messages)",
			Tags:     "text,smoke",
			Provider: "anthropic",
			Method:   "POST",
			Target:   "https://api.anthropic.com/v1/messages",
			Body:     `{"model":"claude-opus-4-8","max_tokens":1024,"stream":true,"messages":[{"role":"user","content":"你是谁?"}]}`,
			Source:   "seed",
		},
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, c := range seeds {
		if _, err := s.db.Exec(
			`INSERT OR IGNORE INTO replay_cases(id, name, tags, provider, method, target, body, source, created_at)
			 VALUES(?,?,?,?,?,?,?,?,?)`,
			c.ID, c.Name, c.Tags, c.Provider, c.Method, c.Target, c.Body, c.Source, now); err != nil {
			return err
		}
	}
	return nil
}
