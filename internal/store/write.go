package store

import (
	"database/sql"
	"fmt"
	"time"

	"tracelab/internal/event"
	"tracelab/internal/sink"
)

// Write persists one captured record: it upserts the session, inserts the event
// spine, the type-specific detail, raw files on disk, sections, FTS text, and
// tags — all in one transaction.
func (s *Store) Write(rec sink.Record) error {
	if rec.Event == nil {
		return fmt.Errorf("store: nil event")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if err := s.writeTx(tx, rec); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *Store) writeTx(tx *sql.Tx, rec sink.Record) error {
	ev := rec.Event
	if err := upsertSession(tx, ev); err != nil {
		return fmt.Errorf("session: %w", err)
	}
	if err := insertEvent(tx, ev); err != nil {
		return fmt.Errorf("event: %w", err)
	}
	switch ev.Kind {
	case event.KindHook:
		if err := s.writeHook(tx, rec); err != nil {
			return fmt.Errorf("hook: %w", err)
		}
	default: // http_exchange / http_tunnel
		if err := s.writeHTTP(tx, rec); err != nil {
			return fmt.Errorf("http: %w", err)
		}
	}
	if err := insertTags(tx, rec); err != nil {
		return fmt.Errorf("tags: %w", err)
	}
	return nil
}

func upsertSession(tx *sql.Tx, ev *event.SourceEvent) error {
	id := ev.Correlation.SessionID
	if id == "" {
		id = "_nosession"
	}
	upstream := ""
	if ev.Capture != nil {
		upstream = originOf(ev.Capture.Target)
	}
	_, err := tx.Exec(
		`INSERT INTO sessions(id, client, upstream, started_at, ended_at)
		 VALUES(?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET ended_at=excluded.ended_at,
		   upstream=COALESCE(NULLIF(sessions.upstream,''), excluded.upstream)`,
		id, ev.Source.Client, upstream, ev.Timing.StartedAt, ev.Timing.CompletedAt)
	return err
}

func insertEvent(tx *sql.Tx, ev *event.SourceEvent) error {
	sessionID := ev.Correlation.SessionID
	if sessionID == "" {
		sessionID = "_nosession"
	}
	_, err := tx.Exec(
		`INSERT INTO events(id, session_id, kind, source_adapter, source_mode, client,
		   turn_id, parent_id, request_id, started_at, completed_at, duration_ms, created_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		ev.ID, sessionID, string(ev.Kind), ev.Source.Adapter, ev.Source.Mode, ev.Source.Client,
		ev.Correlation.TurnID, ev.Correlation.ParentID, ev.Correlation.RequestID,
		ev.Timing.StartedAt, ev.Timing.CompletedAt, ev.Timing.DurationMS,
		time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
