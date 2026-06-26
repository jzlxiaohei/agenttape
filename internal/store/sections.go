package store

import (
	"database/sql"

	"github.com/jzlxiaohei/agenttape/internal/normalize"
)

func insertSections(tx *sql.Tx, eventID string, sections []normalize.SectionStat) error {
	for _, s := range sections {
		if _, err := tx.Exec(
			`INSERT INTO sections(event_id, name, bytes, approx_tokens) VALUES(?,?,?,?)`,
			eventID, s.Name, s.Bytes, s.ApproxTokens); err != nil {
			return err
		}
	}
	return nil
}
