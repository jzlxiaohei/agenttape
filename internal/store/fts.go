package store

import (
	"database/sql"
	"strings"

	"github.com/jzlxiaohei/agenttape/internal/normalize"
)

// insertFTS indexes the meaningful prompt sections for full-text search:
// system, all message text, the final response text, and tool call arguments +
// tool names (next.md 3.2, decision 4).
func insertFTS(tx *sql.Tx, eventID string, env *normalize.NormalizedEnvelope) error {
	var systemText, messagesText, finalText, toolArgs strings.Builder
	if env.Request != nil {
		collectText(&systemText, env.Request.System)
		for _, m := range env.Request.Messages {
			collectText(&messagesText, m.Content)
			collectToolArgs(&toolArgs, m.Content)
		}
		for _, t := range env.Request.Tools {
			toolArgs.WriteString(t.Name)
			toolArgs.WriteByte(' ')
		}
	}
	if env.Response != nil {
		finalText.WriteString(env.Response.FinalText)
		for _, m := range env.Response.Output {
			collectToolArgs(&toolArgs, m.Content)
		}
	}
	_, err := tx.Exec(
		`INSERT INTO events_fts(event_id, system_text, messages_text, final_text, tool_args_text)
		 VALUES(?,?,?,?,?)`,
		eventID, systemText.String(), messagesText.String(), finalText.String(), toolArgs.String())
	return err
}

func collectText(b *strings.Builder, blocks []normalize.ContentBlock) {
	for _, blk := range blocks {
		if blk.Text != "" {
			b.WriteString(blk.Text)
			b.WriteByte('\n')
		}
		if blk.ToolResult != nil {
			collectText(b, blk.ToolResult.Content)
		}
	}
}

func collectToolArgs(b *strings.Builder, blocks []normalize.ContentBlock) {
	for _, blk := range blocks {
		if blk.ToolCall != nil {
			b.WriteString(blk.ToolCall.Name)
			b.WriteByte(' ')
			b.Write(blk.ToolCall.Arguments)
			b.WriteByte('\n')
		}
	}
}
