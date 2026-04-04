package mcpserver

import (
	"testing"
)

func TestGetServer(t *testing.T) {
	t.Parallel()

	m := NewManagerFromServers(map[string]*Server{
		"alpha": {},
		"bravo": {},
	})

	t.Run("existing server", func(t *testing.T) {
		t.Parallel()
		s, err := m.GetServer("alpha")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s == nil {
			t.Fatal("expected non-nil server")
		}
	})

	t.Run("unknown server", func(t *testing.T) {
		t.Parallel()
		_, err := m.GetServer("nonexistent")
		if err == nil {
			t.Fatal("expected error for unknown server")
		}
	})
}

func TestListServerNames(t *testing.T) {
	t.Parallel()

	t.Run("returns all names", func(t *testing.T) {
		t.Parallel()
		m := NewManagerFromServers(map[string]*Server{
			"charlie": {},
			"alpha":   {},
			"bravo":   {},
		})

		names := m.ListServerNames()
		if len(names) != 3 {
			t.Fatalf("got %d names, want 3", len(names))
		}
		nameSet := map[string]bool{}
		for _, n := range names {
			nameSet[n] = true
		}
		for _, expected := range []string{"alpha", "bravo", "charlie"} {
			if !nameSet[expected] {
				t.Errorf("missing expected name %q", expected)
			}
		}
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		m := NewManagerFromServers(map[string]*Server{})
		names := m.ListServerNames()
		if len(names) != 0 {
			t.Errorf("expected empty, got %v", names)
		}
	})
}

func TestManagerClose(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	session := startFakeServer(t, ctx, "tool")
	m := NewManagerFromServers(map[string]*Server{"s": NewServerFromSession(session)})

	// Should not panic on multiple closes.
	m.Close()
	m.Close()
}
