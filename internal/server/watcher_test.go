package server

import (
	"context"
	"testing"
	"time"
)

func TestWatcherResyncOnReconnect(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{}
	seedEnvironment(t, srv)

	env, err := srv.queries.GetEnvironment(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetEnvironment: %v", err)
	}

	sub, unsubscribe := srv.events.Subscribe()
	defer unsubscribe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signals := make(chan struct{})
	errc := make(chan error)
	go srv.consumeEvents(ctx, env, signals, errc, false)

	select {
	case evt := <-sub:
		if evt.Type != "environment.updated" {
			t.Fatalf("resync: want environment.updated, got %q", evt.Type)
		}
	case <-time.After(watcherConnectGrace + 2*time.Second):
		t.Fatal("expected a resync publish shortly after reconnect")
	}
}

func TestWatcherFirstConnectDoesNotResync(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{}
	seedEnvironment(t, srv)

	env, err := srv.queries.GetEnvironment(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetEnvironment: %v", err)
	}

	sub, unsubscribe := srv.events.Subscribe()
	defer unsubscribe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signals := make(chan struct{})
	errc := make(chan error)
	go srv.consumeEvents(ctx, env, signals, errc, true)

	select {
	case evt := <-sub:
		t.Fatalf("first connect should not resync, got %q", evt.Type)
	case <-time.After(watcherConnectGrace + time.Second):
	}
}
