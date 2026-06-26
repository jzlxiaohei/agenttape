package store

import (
	"fmt"

	"agenttape/internal/source/hook"
)

// HookClients is the closed set of clients whose hook event set is configurable.
// Keyed by the runtime value stamped on /_hook posts (so config lines up with
// captured events). Codex CLI and Codex desktop share the single "codex" key —
// they wire the same event set, just via different injection mechanisms.
var HookClients = []string{"claude_code", "codex"}

// HookEventDef is one configurable hook event for a client: whether agenttape
// wires it on launch, and whether it is a built-in default or user-added.
type HookEventDef struct {
	Client  string `json:"client"`
	Event   string `json:"event"`
	Enabled bool   `json:"enabled"`
	Source  string `json:"source"` // seed | user
}

// ClientHookEvents returns the ENABLED event names for a client, alphabetical.
// This is what the launcher injects; an empty result means the user has disabled
// every event (a legitimate choice — capture nothing for that client).
func (s *Store) ClientHookEvents(client string) ([]string, error) {
	rows, err := s.db.Query(
		`SELECT event FROM client_hook_events
		 WHERE client = ? AND enabled = 1 ORDER BY event`, client)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var ev string
		if err := rows.Scan(&ev); err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

// ListHookEvents returns every configured event across all clients, ordered by
// client then event, for the management UI.
func (s *Store) ListHookEvents() ([]HookEventDef, error) {
	rows, err := s.db.Query(
		`SELECT client, event, enabled, source FROM client_hook_events
		 ORDER BY client, event`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HookEventDef{}
	for rows.Next() {
		var d HookEventDef
		if err := rows.Scan(&d.Client, &d.Event, &d.Enabled, &d.Source); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// AddHookEvent registers a user-added event (enabled). Re-adding an existing
// event just re-enables it without clobbering its source (a re-added seed stays
// a seed). Names are not validated: unknown events are harmlessly ignored by the
// runtime, so adding one a release ahead of time is safe and intended.
func (s *Store) AddHookEvent(client, event string) error {
	_, err := s.db.Exec(
		`INSERT INTO client_hook_events(client, event, enabled, source)
		 VALUES(?,?,1,'user')
		 ON CONFLICT(client, event) DO UPDATE SET enabled = 1`,
		client, event)
	return err
}

// SetHookEventEnabled toggles whether an event is wired on launch. Used to mute
// a noisy default without deleting it (so an upgrade can't silently re-add it).
func (s *Store) SetHookEventEnabled(client, event string, enabled bool) error {
	res, err := s.db.Exec(
		`UPDATE client_hook_events SET enabled = ? WHERE client = ? AND event = ?`,
		enabled, client, event)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNoRows
	}
	return nil
}

// DeleteHookEvent removes a user-added event. Seed (built-in) events are
// protected at the handler layer — disable them instead — so this stays a plain
// delete usable by tests.
func (s *Store) DeleteHookEvent(client, event string) error {
	_, err := s.db.Exec(
		`DELETE FROM client_hook_events WHERE client = ? AND event = ?`, client, event)
	return err
}

// seedClientHookEvents installs the built-in default event set for each client
// once (INSERT OR IGNORE so user toggles/additions survive restart, exactly like
// seedCases). The defaults live in package hook (canonical, tracked against each
// runtime's docs); the store is the single source read at launch.
func (s *Store) seedClientHookEvents() error {
	defaults := map[string][]string{
		"claude_code": hook.DefaultClaudeEvents(),
		"codex":       hook.DefaultCodexEvents(),
	}
	for _, client := range HookClients {
		for _, ev := range defaults[client] {
			if _, err := s.db.Exec(
				`INSERT OR IGNORE INTO client_hook_events(client, event, enabled, source)
				 VALUES(?,?,1,'seed')`, client, ev); err != nil {
				return fmt.Errorf("seed hook event %s/%s: %w", client, ev, err)
			}
		}
	}
	return nil
}
