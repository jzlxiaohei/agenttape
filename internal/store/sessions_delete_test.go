package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"tracelab/internal/store"
)

func hasSession(ss []store.SessionSummary, id string) bool {
	for _, s := range ss {
		if s.ID == id {
			return true
		}
	}
	return false
}

// TestDeleteSessionCascades verifies deleting a session removes its events, the
// per-type detail/child rows, and the raw bytes on disk — leaving nothing behind.
func TestDeleteSessionCascades(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	if err := st.Write(httpRecord(t, "sess-del")); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Sanity: the session, its events, and a raw file all exist.
	if ss, _ := st.ListSessions(); !hasSession(ss, "sess-del") {
		t.Fatal("session missing before delete")
	}
	if ev, _ := st.ListEvents("sess-del"); len(ev) == 0 {
		t.Fatal("events missing before delete")
	}
	rawDir := filepath.Join(dir, "raw", "sess-del")
	if _, err := os.Stat(rawDir); err != nil {
		t.Fatalf("raw dir missing before delete: %v", err)
	}

	if err := st.DeleteSession("sess-del"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if ss, _ := st.ListSessions(); hasSession(ss, "sess-del") {
		t.Error("session still listed after delete")
	}
	if ev, _ := st.ListEvents("sess-del"); len(ev) != 0 {
		t.Errorf("events remain after delete: %d", len(ev))
	}
	if _, err := os.Stat(rawDir); !os.IsNotExist(err) {
		t.Errorf("raw dir not removed (stat err = %v)", err)
	}
	// Child rows must be gone too (raw_files is representative).
	var n int
	if err := st.DB().QueryRow(`SELECT COUNT(*) FROM raw_files`).Scan(&n); err != nil {
		t.Fatalf("count raw_files: %v", err)
	}
	if n != 0 {
		t.Errorf("raw_files rows remain after delete: %d", n)
	}
}
