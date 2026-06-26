package store

import (
	"database/sql"

	"agenttape/internal/sink"
)

func (s *Store) writeHook(tx *sql.Tx, rec sink.Record) error {
	ev := rec.Event
	h := ev.Hook
	runtime, name, toolCallID, toolName := "", "", "", ""
	if h != nil {
		runtime, name, toolCallID, toolName = h.Runtime, h.EventName, h.ToolCallID, h.ToolName
	}
	if _, err := tx.Exec(
		`INSERT INTO hook_events(event_id, runtime, event_name, tool_call_id, tool_name)
		 VALUES(?,?,?,?,?)`,
		ev.ID, runtime, name, toolCallID, toolName); err != nil {
		return err
	}
	return s.writeRawFiles(tx, ev)
}
