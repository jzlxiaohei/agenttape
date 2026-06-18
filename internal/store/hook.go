package store

import (
	"database/sql"

	"tracelab/internal/sink"
)

func (s *Store) writeHook(tx *sql.Tx, rec sink.Record) error {
	ev := rec.Event
	h := ev.Hook
	runtime, name, toolCallID := "", "", ""
	if h != nil {
		runtime, name, toolCallID = h.Runtime, h.EventName, h.ToolCallID
	}
	if _, err := tx.Exec(
		`INSERT INTO hook_events(event_id, runtime, event_name, tool_call_id)
		 VALUES(?,?,?,?)`,
		ev.ID, runtime, name, toolCallID); err != nil {
		return err
	}
	return s.writeRawFiles(tx, ev)
}
