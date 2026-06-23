package store

import (
	"database/sql"
	"encoding/json"
	"strings"

	"tracelab/internal/event"
	"tracelab/internal/normalize"
	"tracelab/internal/sink"
)

func (s *Store) writeHTTP(tx *sql.Tx, rec sink.Record) error {
	ev := rec.Event
	cap := ev.Capture
	if cap == nil {
		return nil
	}
	env := rec.Normalized

	var provider, model, wireAPI, normJSON string
	var in, out, total, cached, reasoning int64
	if env != nil {
		provider, model, wireAPI = env.Provider.Name, env.Provider.Model, env.Provider.WireAPI
		if env.Response != nil && env.Response.Usage != nil {
			u := env.Response.Usage
			in, out, total = u.InputTokens, u.OutputTokens, u.TotalTokens
			cached = extraInt(u.Extra, "cached_tokens")
			reasoning = extraInt(u.Extra, "reasoning_tokens")
		}
		stripBlockRaw(env)
		b, _ := json.Marshal(env)
		normJSON = string(b)
	}

	if _, err := tx.Exec(
		`INSERT INTO http_exchanges(event_id, method, url, target, response_status,
		   is_completion, provider, model, wire_api, normalize_error,
		   input_tokens, output_tokens, total_tokens, cached_tokens, reasoning_tokens,
		   normalized_json)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		ev.ID, cap.Method, cap.URL, cap.Target, cap.Response.StatusCode,
		boolInt(isCompletion(ev, env)), provider, model, wireAPI, rec.NormalizeError,
		in, out, total, cached, reasoning, normJSON); err != nil {
		return err
	}

	if err := s.writeRawFiles(tx, ev); err != nil {
		return err
	}
	if env != nil && env.Request != nil {
		if err := insertSections(tx, ev.ID, env.Request.Sections); err != nil {
			return err
		}
	}
	if env != nil {
		if err := insertFTS(tx, ev.ID, env); err != nil {
			return err
		}
	}
	// Derive a human-friendly session title from the first user prompt, set once
	// (the first completion to land wins; later ones leave it untouched).
	if isCompletion(ev, env) {
		if title := sessionTitle(env); title != "" {
			sid := ev.Correlation.SessionID
			if sid == "" {
				sid = "_nosession"
			}
			if _, err := tx.Exec(
				`UPDATE sessions SET label=? WHERE id=? AND (label IS NULL OR label='')`,
				title, sid); err != nil {
				return err
			}
		}
	}
	return nil
}

// isCompletion distinguishes a real model call from a control/probe request.
func isCompletion(ev *event.SourceEvent, env *normalize.NormalizedEnvelope) bool {
	if env == nil || ev.Capture == nil {
		return false
	}
	return ev.Capture.Method == "POST" && env.Request != nil
}

func extraInt(m map[string]any, key string) int64 {
	switch v := m[key].(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	}
	return 0
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// originOf returns scheme://host for a URL, used as a session's upstream hint.
func originOf(u string) string {
	i := strings.Index(u, "://")
	if i < 0 {
		return u
	}
	rest := u[i+3:]
	if j := strings.IndexByte(rest, '/'); j >= 0 {
		return u[:i+3+j]
	}
	return u
}
