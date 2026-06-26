package store_test

import (
	"slices"
	"testing"

	"agenttape/internal/source/hook"
	"agenttape/internal/store"
)

func TestClientHookEvents(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(dir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	// Open() seeds the built-in defaults; every default starts enabled.
	got, err := st.ClientHookEvents("claude_code")
	if err != nil {
		t.Fatalf("ClientHookEvents: %v", err)
	}
	if !slices.Equal(sortedCopy(got), sortedCopy(hook.DefaultClaudeEvents())) {
		t.Fatalf("seeded claude events = %v, want defaults %v", got, hook.DefaultClaudeEvents())
	}

	// Disabling a default drops it from the launch set without deleting it.
	if err := st.SetHookEventEnabled("claude_code", "Notification", false); err != nil {
		t.Fatalf("disable: %v", err)
	}
	got, _ = st.ClientHookEvents("claude_code")
	if slices.Contains(got, "Notification") {
		t.Error("disabled event should not be in the enabled set")
	}

	// A user-added event shows up enabled.
	if err := st.AddHookEvent("claude_code", "BrandNewEvent"); err != nil {
		t.Fatalf("add: %v", err)
	}
	got, _ = st.ClientHookEvents("claude_code")
	if !slices.Contains(got, "BrandNewEvent") {
		t.Error("added event missing from enabled set")
	}

	// Source is tracked: seed vs user (drives delete protection in the handler).
	defs, err := st.ListHookEvents()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if sourceOf(defs, "claude_code", "BrandNewEvent") != "user" {
		t.Error("added event should have source=user")
	}
	if sourceOf(defs, "claude_code", "SessionStart") != "seed" {
		t.Error("default event should have source=seed")
	}

	// Deleting the user event removes it.
	if err := st.DeleteHookEvent("claude_code", "BrandNewEvent"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, _ = st.ClientHookEvents("claude_code")
	if slices.Contains(got, "BrandNewEvent") {
		t.Error("deleted event still present")
	}

	// codex is seeded independently.
	codex, _ := st.ClientHookEvents("codex")
	if len(codex) == 0 {
		t.Error("codex events should be seeded too")
	}
}

func sortedCopy(s []string) []string {
	out := append([]string(nil), s...)
	slices.Sort(out)
	return out
}

func sourceOf(defs []store.HookEventDef, client, event string) string {
	for _, d := range defs {
		if d.Client == client && d.Event == event {
			return d.Source
		}
	}
	return ""
}
