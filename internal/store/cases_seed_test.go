package store_test

import (
	"strings"
	"testing"

	"tracelab/internal/store"
)

// TestSeedCasesContainNoProjectConversation keeps built-in material portable:
// seeds may preserve real wire shapes, but must not embed a conversation about
// TraceLab itself, local paths, or discarded product language.
func TestSeedCasesContainNoProjectConversation(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	cases, err := st.ListCases()
	if err != nil {
		t.Fatalf("list cases: %v", err)
	}
	forbidden := []string{
		"tracelab",
		"worklog.md",
		"replay_lib.md",
		"deeper_viewer.md",
		"aethertrace",
		"/users/",
		"降智",
	}
	for _, c := range cases {
		if c.Source != "seed" {
			continue
		}
		body := strings.ToLower(c.Body)
		for _, term := range forbidden {
			if strings.Contains(body, term) {
				t.Errorf("seed %s contains forbidden project-conversation term %q", c.ID, term)
			}
		}
	}
}

func hasCase(cases []store.ReplayCase, id string) bool {
	for _, c := range cases {
		if c.ID == id {
			return true
		}
	}
	return false
}

// TestSeedCasesOnce verifies that when the embedded seed set is unchanged (digest
// matches), seeding is a no-op: a deleted built-in case stays deleted across a
// restart instead of resurrecting.
func TestSeedCasesOnce(t *testing.T) {
	dir := t.TempDir()

	st, err := store.Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	cases, _ := st.ListCases()
	if !hasCase(cases, "seed:codex-pure-text") {
		t.Fatal("expected seed case on first open")
	}
	if err := st.DeleteCase("seed:codex-pure-text"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	st.Close()

	// Reopen the same database — seeding must NOT re-add the deleted built-in.
	st2, err := store.Open(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer st2.Close()
	cases2, _ := st2.ListCases()
	if hasCase(cases2, "seed:codex-pure-text") {
		t.Error("deleted built-in case resurrected after restart")
	}
	if !hasCase(cases2, "seed:cc-pure-text") {
		t.Error("an untouched built-in case should survive")
	}
}

// TestSeedCasesRefreshOnSeedChange verifies that when the embedded seed set changes
// (detected via the content digest in schema_meta), built-in rows are reinstalled —
// even ones the user had deleted — while user-made cases are left untouched.
func TestSeedCasesRefreshOnSeedChange(t *testing.T) {
	dir := t.TempDir()

	st, err := store.Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	// A user's own case must survive a seed refresh (different id namespace).
	if err := st.AddCase(store.ReplayCase{
		ID: "user:mine", Name: "mine", Provider: "anthropic",
		Method: "POST", Endpoint: "/v1/messages", Body: "{}", Source: "manual",
	}); err != nil {
		t.Fatalf("add user case: %v", err)
	}
	if err := st.DeleteCase("seed:codex-pure-text"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	// Simulate the embedded seeds changing since last run by busting the stored digest;
	// the next Open must see a mismatch and reinstall the full built-in set.
	if _, err := st.DB().Exec(
		`UPDATE schema_meta SET value = 'stale' WHERE key = 'cases_seed_digest'`); err != nil {
		t.Fatalf("bust digest: %v", err)
	}
	st.Close()

	st2, err := store.Open(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer st2.Close()
	cases, _ := st2.ListCases()
	if !hasCase(cases, "seed:codex-pure-text") {
		t.Error("a seed-set change should reinstall a previously deleted built-in")
	}
	if !hasCase(cases, "user:mine") {
		t.Error("user-made case must survive a seed refresh")
	}
}
