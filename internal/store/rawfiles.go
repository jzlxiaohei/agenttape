package store

import (
	"database/sql"
	"os"
	"path/filepath"

	"agenttape/internal/event"
)

// roleSuffix maps an artifact role to a human-friendly filename suffix so users
// can browse the raw/ directory and open files with their own tools.
var roleSuffix = map[event.ArtifactRole]string{
	event.RoleRequestBody:  "request.json",
	event.RoleResponseBody: "response.txt",
	event.RoleHookPayload:  "hook.json",
}

// writeRawFiles writes each raw artifact to disk under raw/<session>/ and records
// a pointer row. Bytes live on the filesystem (design choice C) and are retained
// as a user-facing feature.
func (s *Store) writeRawFiles(tx *sql.Tx, ev *event.SourceEvent) error {
	session := ev.Correlation.SessionID
	if session == "" {
		session = "_nosession"
	}
	dir := filepath.Join(s.rawDir, session)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for i := range ev.RawArtifacts {
		a := &ev.RawArtifacts[i]
		body, err := a.Bytes()
		if err != nil {
			continue // keep going; one bad artifact must not drop the record
		}
		suffix := roleSuffix[a.Role]
		if suffix == "" {
			suffix = "body.bin"
		}
		fname := ev.ID + "." + suffix
		abs := filepath.Join(dir, fname)
		if err := os.WriteFile(abs, body, 0o644); err != nil {
			return err
		}
		rel := filepath.Join("raw", session, fname)
		if _, err := tx.Exec(
			`INSERT INTO raw_files(id, event_id, role, path, media_type,
			   content_encoding, body_encoding, size_bytes, redaction_applied)
			 VALUES(?,?,?,?,?,?,?,?,?)`,
			a.ID, ev.ID, string(a.Role), rel, a.MediaType,
			a.ContentEncoding, string(a.BodyEncoding), a.SizeBytes, boolInt(a.RedactionApplied)); err != nil {
			return err
		}
	}
	return nil
}
