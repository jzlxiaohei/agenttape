package store

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Built-in case bodies, shipped as files so the realistic (and large) wire shapes
// stay readable and diffable. Each is a genuine request shape — derived from real
// captured codex/claude traffic, then stripped of credentials, identifiers, and any
// user content — so it is accepted by upstream instead of 400'd like a toy payload.
// They map to the replay-note experiments: pure text, a tool read round-trip, a file
// edit, and a failed tool result, for both wire formats (plus a CC session-title
// generation request and the original full-shape CC message).
//
//go:embed seeds/cc-full-messages.json
var ccFullMessagesBody string

//go:embed seeds/codex-pure-text.json
var codexPureTextBody string

//go:embed seeds/codex-tool-read.json
var codexToolReadBody string

//go:embed seeds/codex-apply-patch.json
var codexApplyPatchBody string

//go:embed seeds/codex-tool-failure.json
var codexToolFailureBody string

//go:embed seeds/cc-pure-text.json
var ccPureTextBody string

//go:embed seeds/cc-tool-read.json
var ccToolReadBody string

//go:embed seeds/cc-edit.json
var ccEditBody string

//go:embed seeds/cc-tool-failure.json
var ccToolFailureBody string

//go:embed seeds/cc-title.json
var ccTitleBody string

//go:embed seeds/cc-compaction.json
var ccCompactionBody string

// ReplayCase is one saved, re-sendable request in the replay library.
type ReplayCase struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Tags      string `json:"tags"`
	Provider  string `json:"provider"`
	Method    string `json:"method"`
	Target    string `json:"target"`
	Endpoint  string `json:"endpoint"`
	Body      string `json:"body"`
	Source    string `json:"source"` // seed | captured | snapshot
	CreatedAt string `json:"created_at"`
}

// ListCases returns all replay-library cases, newest first.
func (s *Store) ListCases() ([]ReplayCase, error) {
	rows, err := s.db.Query(
		`SELECT id, name, tags, provider, method, target, COALESCE(endpoint,''), body, source, created_at
		 FROM replay_cases ORDER BY created_at DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReplayCase{}
	for rows.Next() {
		var c ReplayCase
		if err := rows.Scan(&c.ID, &c.Name, &c.Tags, &c.Provider, &c.Method, &c.Target, &c.Endpoint, &c.Body, &c.Source, &c.CreatedAt); err != nil {
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
		`SELECT id, name, tags, provider, method, target, COALESCE(endpoint,''), body, source, created_at
		 FROM replay_cases WHERE id = ?`, id).Scan(
		&c.ID, &c.Name, &c.Tags, &c.Provider, &c.Method, &c.Target, &c.Endpoint, &c.Body, &c.Source, &c.CreatedAt)
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
	if c.Endpoint == "" {
		c.Endpoint = EndpointForTarget(c.Provider, c.Target, "")
	}
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO replay_cases(id, name, tags, provider, method, target, endpoint, body, source, created_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?)`,
		c.ID, c.Name, c.Tags, c.Provider, c.Method, c.Target, c.Endpoint, c.Body, c.Source, c.CreatedAt)
	return err
}

// DeleteCase removes a case by id. Seed (built-in) cases are protected at the
// handler layer, not here, so this stays a plain delete usable by tests.
func (s *Store) DeleteCase(id string) error {
	_, err := s.db.Exec(`DELETE FROM replay_cases WHERE id = ?`, id)
	return err
}

// seedCases installs the built-in cases and keeps them in sync with the embedded
// seed files. It fingerprints the embedded seed set (every seed's fields + body) and
// stores the digest in schema_meta under 'cases_seed_digest'. When the digest is
// unchanged it does nothing — so for a released binary the built-in set is stable and
// a user's in-place edits/deletions of built-ins persist across restarts. When the
// embedded seeds change (a rebuild after editing seeds/*.json, or a new agenttape
// version that adds/edits seeds) the digest differs and every seed row is reinstalled
// via INSERT OR REPLACE — no manual SQL reset needed.
//
// Trade-off: a seed-set change reinstalls the FULL built-in set, so any built-in a
// user deleted or overwrote in place comes back (refreshed) on that rebuild. Self-made
// cases are never touched — INSERT OR REPLACE only ever hits namespaced seed:* ids.
func (s *Store) seedCases() error {
	const (
		codexTarget = "https://chatgpt.com/backend-api/codex/responses"
		ccTarget    = "https://api.anthropic.com/v1/messages"
	)
	codexCase := func(id, name, tags, body string) ReplayCase {
		return ReplayCase{ID: id, Name: name, Tags: tags, Provider: "openai-responses",
			Method: "POST", Target: codexTarget, Endpoint: "/responses", Body: body, Source: "seed"}
	}
	ccCase := func(id, name, tags, body string) ReplayCase {
		return ReplayCase{ID: id, Name: name, Tags: tags, Provider: "anthropic",
			Method: "POST", Target: ccTarget, Endpoint: "/v1/messages", Body: body, Source: "seed"}
	}
	// Names here are English fallbacks; the UI shows a localized title keyed by the
	// seed id (cases.seed.<id>) so built-in titles aren't hardcoded to one language.
	seeds := []ReplayCase{
		// codex (openai-responses) — note experiments 1–4
		codexCase("seed:codex-pure-text", "Plain text · no tools (codex)", "text,smoke,codex", codexPureTextBody),
		codexCase("seed:codex-tool-read", "Tool round-trip · read file (codex)", "tool,codex", codexToolReadBody),
		codexCase("seed:codex-apply-patch", "Edit file · apply_patch (codex)", "tool,edit,codex", codexApplyPatchBody),
		codexCase("seed:codex-tool-failure", "Tool failure · error feedback (codex)", "tool,error,codex", codexToolFailureBody),
		// claude code (anthropic) — note experiments 1, 2, 4 + extras
		ccCase("seed:cc-pure-text", "Plain text · no tools (cc)", "text,smoke,cc", ccPureTextBody),
		ccCase("seed:cc-tool-read", "Tool round-trip · read file (cc)", "tool,cc", ccToolReadBody),
		// experiment 3 for cc: a structured Edit round-trip — cc's analog to codex's
		// freeform apply_patch (cc has no apply_patch; it edits via Edit/Write).
		ccCase("seed:cc-edit", "Edit file · structured Edit (cc)", "tool,edit,cc", ccEditBody),
		ccCase("seed:cc-tool-failure", "Tool failure · error feedback (cc)", "tool,error,cc", ccToolFailureBody),
		ccCase("seed:cc-title", "Session title generation (cc)", "text,cc", ccTitleBody),
		ccCase("seed:cc-full-claude", "Full request shape (cc)", "text,cc,smoke", ccFullMessagesBody),
		// Experiment 5: a synthetic conversation retaining the real /compact wire
		// shape — the request asks the model to summarize into <analysis>/<summary>.
		ccCase("seed:cc-compaction", "Compaction · summarize conversation (cc)", "compaction,context,cc", ccCompactionBody),
	}
	digest := seedDigest(seeds)
	var stored string
	if err := s.db.QueryRow(
		`SELECT value FROM schema_meta WHERE key = 'cases_seed_digest'`).Scan(&stored); err == nil && stored == digest {
		return nil // embedded seed set unchanged since last run
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, c := range seeds {
		if _, err := s.db.Exec(
			`INSERT OR REPLACE INTO replay_cases(id, name, tags, provider, method, target, endpoint, body, source, created_at)
			 VALUES(?,?,?,?,?,?,?,?,?,?)`,
			c.ID, c.Name, c.Tags, c.Provider, c.Method, c.Target, c.Endpoint, c.Body, c.Source, now); err != nil {
			return err
		}
	}
	if _, err := s.db.Exec(
		`INSERT INTO schema_meta(key, value) VALUES('cases_seed_digest', ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, digest); err != nil {
		return err
	}
	// Retire the pre-digest once-per-database flag; the digest supersedes it.
	_, err := s.db.Exec(`DELETE FROM schema_meta WHERE key = 'cases_seeded'`)
	return err
}

// seedDigest fingerprints the embedded seed set so seedCases can detect when the
// built-in definitions change (a rebuild) and reinstall them. It hashes every field
// that gets written, and sorts the per-seed lines first so merely reordering the
// seeds slice doesn't churn the digest.
func seedDigest(seeds []ReplayCase) string {
	lines := make([]string, len(seeds))
	for i, c := range seeds {
		lines[i] = strings.Join([]string{
			c.ID, c.Name, c.Tags, c.Provider, c.Method, c.Target, c.Endpoint, c.Body,
		}, "\x00")
	}
	sort.Strings(lines)
	h := sha256.New()
	for _, l := range lines {
		h.Write([]byte(l))
		h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// EndpointForTarget turns an absolute captured target into the path that should
// be replayed through a launch session. The session owns the upstream; the case
// owns only the request shape and endpoint.
func EndpointForTarget(provider, target, upstream string) string {
	cleanUpstream := strings.TrimRight(upstream, "/")
	if cleanUpstream != "" && strings.HasPrefix(target, cleanUpstream+"/") {
		return ensureSlash(strings.TrimPrefix(target, cleanUpstream))
	}

	u, err := url.Parse(target)
	if err != nil {
		return ensureSlash(target)
	}
	path := u.EscapedPath()
	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}

	p := strings.ToLower(provider)
	if strings.Contains(p, "openai") && strings.HasSuffix(u.EscapedPath(), "/responses") {
		return "/responses"
	}
	if strings.Contains(p, "anthropic") && strings.Contains(u.EscapedPath(), "/v1/messages") {
		return "/v1/messages"
	}
	return ensureSlash(path)
}

func ensureSlash(s string) string {
	if s == "" || s[0] == '/' {
		return s
	}
	return "/" + s
}
