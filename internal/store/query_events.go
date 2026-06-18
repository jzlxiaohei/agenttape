package store

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
)

// EventSummary is a row in a session's event timeline.
type EventSummary struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	StartedAt      string `json:"started_at"`
	Method         string `json:"method"`
	Target         string `json:"target"`
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	IsCompletion   bool   `json:"is_completion"`
	ResponseStatus int    `json:"response_status"`
	TotalTokens    int64  `json:"total_tokens"`
}

// ListEvents returns a session's events in chronological order (http + hook
// interleaved on the shared spine).
func (s *Store) ListEvents(sessionID string) ([]EventSummary, error) {
	rows, err := s.db.Query(
		`SELECT e.id, e.kind, e.started_at,
		   COALESCE(h.method,''), COALESCE(h.target,''), COALESCE(h.provider,''),
		   COALESCE(h.model,''), COALESCE(h.is_completion,0),
		   COALESCE(h.response_status,0), COALESCE(h.total_tokens,0)
		 FROM events e LEFT JOIN http_exchanges h ON h.event_id = e.id
		 WHERE e.session_id = ? ORDER BY e.started_at, e.created_at`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EventSummary
	for rows.Next() {
		var e EventSummary
		var isComp int
		if err := rows.Scan(&e.ID, &e.Kind, &e.StartedAt, &e.Method, &e.Target,
			&e.Provider, &e.Model, &isComp, &e.ResponseStatus, &e.TotalTokens); err != nil {
			return nil, err
		}
		e.IsCompletion = isComp == 1
		out = append(out, e)
	}
	return out, rows.Err()
}

// TagInfo / RawFileInfo describe an event's tags and raw artifacts.
type TagInfo struct {
	Tag        string  `json:"tag"`
	Confidence float64 `json:"confidence"`
	Suspected  bool    `json:"suspected"`
	Source     string  `json:"source"`
	Evidence   string  `json:"evidence"`
}

type RawFileInfo struct {
	Role      string `json:"role"`
	MediaType string `json:"media_type"`
	SizeBytes int64  `json:"size_bytes"`
}

// EventDetail is the full per-event view for the detail screen.
type EventDetail struct {
	ID             string          `json:"id"`
	Kind           string          `json:"kind"`
	SessionID      string          `json:"session_id"`
	StartedAt      string          `json:"started_at"`
	CompletedAt    string          `json:"completed_at"`
	DurationMS     int64           `json:"duration_ms"`
	Method         string          `json:"method"`
	Target         string          `json:"target"`
	ResponseStatus int             `json:"response_status"`
	Provider       string          `json:"provider"`
	Model          string          `json:"model"`
	IsCompletion   bool            `json:"is_completion"`
	NormalizeError string          `json:"normalize_error,omitempty"`
	Normalized     json.RawMessage `json:"normalized,omitempty"`
	Tags           []TagInfo       `json:"tags"`
	RawFiles       []RawFileInfo   `json:"raw_files"`
}

// GetEvent assembles the full detail for one event. Returns sql.ErrNoRows if the
// event does not exist.
func (s *Store) GetEvent(id string) (*EventDetail, error) {
	d := &EventDetail{Tags: []TagInfo{}, RawFiles: []RawFileInfo{}}
	var isComp int
	var normJSON string
	err := s.db.QueryRow(
		`SELECT e.id, e.kind, e.session_id, e.started_at, e.completed_at, e.duration_ms,
		   COALESCE(h.method,''), COALESCE(h.target,''), COALESCE(h.response_status,0),
		   COALESCE(h.provider,''), COALESCE(h.model,''), COALESCE(h.is_completion,0),
		   COALESCE(h.normalize_error,''), COALESCE(h.normalized_json,'')
		 FROM events e LEFT JOIN http_exchanges h ON h.event_id = e.id
		 WHERE e.id = ?`, id).Scan(
		&d.ID, &d.Kind, &d.SessionID, &d.StartedAt, &d.CompletedAt, &d.DurationMS,
		&d.Method, &d.Target, &d.ResponseStatus, &d.Provider, &d.Model, &isComp,
		&d.NormalizeError, &normJSON)
	if err != nil {
		return nil, err
	}
	d.IsCompletion = isComp == 1
	if normJSON != "" {
		d.Normalized = json.RawMessage(normJSON)
	}
	if err := s.loadTags(d); err != nil {
		return nil, err
	}
	return d, s.loadRawFiles(d)
}

func (s *Store) loadTags(d *EventDetail) error {
	rows, err := s.db.Query(
		`SELECT tag, confidence, suspected, source, evidence FROM tags WHERE event_id = ?`, d.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var t TagInfo
		var suspected int
		if err := rows.Scan(&t.Tag, &t.Confidence, &suspected, &t.Source, &t.Evidence); err != nil {
			return err
		}
		t.Suspected = suspected == 1
		d.Tags = append(d.Tags, t)
	}
	return rows.Err()
}

func (s *Store) loadRawFiles(d *EventDetail) error {
	rows, err := s.db.Query(
		`SELECT role, media_type, size_bytes FROM raw_files WHERE event_id = ?`, d.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var f RawFileInfo
		if err := rows.Scan(&f.Role, &f.MediaType, &f.SizeBytes); err != nil {
			return err
		}
		d.RawFiles = append(d.RawFiles, f)
	}
	return rows.Err()
}

// RawFilePath returns the absolute path of an event's raw artifact for a role.
func (s *Store) RawFilePath(eventID, role string) (string, error) {
	var rel string
	err := s.db.QueryRow(
		`SELECT path FROM raw_files WHERE event_id = ? AND role = ?`, eventID, role).Scan(&rel)
	if err != nil {
		return "", err
	}
	return filepath.Join(s.dataDir, rel), nil
}

// ErrNoRows is re-exported so handlers can map missing events to 404 without
// importing database/sql.
var ErrNoRows = sql.ErrNoRows
