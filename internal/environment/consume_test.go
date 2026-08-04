package environment

import (
	"context"
	"testing"
	"time"
)

func TestReconnectTriggersAResync(t *testing.T) {
	h := newWatcherHarness(t, fakeDocker{}.factory())
	h.seedEnvironment(t)

	env, err := h.queries.GetEnvironment(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetEnvironment: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go h.consume(ctx, env, make(chan struct{}), make(chan error), false)

	select {
	case <-h.observed:
	case <-time.After(watcherConnectGrace + 2*time.Second):
		t.Fatal("a reconnect must republish the environment, in case events were missed while disconnected")
	}
}

func TestFirstConnectDoesNotResync(t *testing.T) {
	h := newWatcherHarness(t, fakeDocker{}.factory())
	h.seedEnvironment(t)

	env, err := h.queries.GetEnvironment(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetEnvironment: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go h.consume(ctx, env, make(chan struct{}), make(chan error), true)

	select {
	case detail := <-h.observed:
		t.Fatalf("the first connect must not resync, got %+v", detail)
	case <-time.After(watcherConnectGrace + time.Second):
	}
}
