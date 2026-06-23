package store_test

import (
	"testing"

	"tracelab/internal/store"
)

func hasCase(cases []store.ReplayCase, id string) bool {
	for _, c := range cases {
		if c.ID == id {
			return true
		}
	}
	return false
}

// TestSeedCasesOnce verifies seeding happens once per database: a deleted built-in
// case stays deleted across a restart instead of resurrecting.
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
