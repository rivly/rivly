package docker

import (
	"context"
	"testing"
)

const missingSocket = "unix:///tmp/rivly-nonexistent.sock"

func TestPingUnreachable(t *testing.T) {
	m := NewManager()
	defer m.Close()

	st := m.Ping(context.Background(), 1, missingSocket)
	if st.Up {
		t.Fatal("expected down for a nonexistent socket")
	}
	if st.Error == "" {
		t.Fatal("expected an error message")
	}
}

func TestInfoUnreachable(t *testing.T) {
	m := NewManager()
	defer m.Close()

	if _, err := m.Info(context.Background(), 1, missingSocket); err == nil {
		t.Fatal("expected an error for a nonexistent socket")
	}
}

func TestClientCached(t *testing.T) {
	m := NewManager()
	defer m.Close()

	c1, err := m.clientFor(1, missingSocket)
	if err != nil {
		t.Fatalf("clientFor: %v", err)
	}
	c2, err := m.clientFor(1, missingSocket)
	if err != nil {
		t.Fatalf("clientFor: %v", err)
	}
	if c1 != c2 {
		t.Fatal("expected the cached client to be reused for the same id")
	}
}
