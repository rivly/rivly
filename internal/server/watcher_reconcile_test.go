package server

import (
	"context"
	"testing"
	"time"
)

func TestWatchersPickUpAnEnvironmentTheyMissedAtStartup(t *testing.T) {
	started := make(chan int64, 4)
	srv := newTestServer(t)
	srv.docker = fakeDocker{watchStarted: started}

	previous := watcherRefresh
	watcherRefresh = 20 * time.Millisecond
	t.Cleanup(func() { watcherRefresh = previous })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		if err := srv.Wait(stopCtx); err != nil {
			t.Errorf("watchers did not stop: %v", err)
		}
	})

	srv.Background(ctx, srv.RunWatchers)

	select {
	case id := <-started:
		t.Fatalf("watched environment %d before any existed", id)
	case <-time.After(100 * time.Millisecond):
	}

	seedEnvironment(t, srv)

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("an environment that appeared after startup was never watched")
	}
}

func TestWatchersDoNotStartTwiceForTheSameEnvironment(t *testing.T) {
	started := make(chan int64, 8)
	srv := newTestServer(t)
	srv.docker = fakeDocker{watchStarted: started}
	seedEnvironment(t, srv)

	previous := watcherRefresh
	watcherRefresh = 10 * time.Millisecond
	t.Cleanup(func() { watcherRefresh = previous })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		if err := srv.Wait(stopCtx); err != nil {
			t.Errorf("watchers did not stop: %v", err)
		}
	})

	srv.Background(ctx, srv.RunWatchers)

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("the seeded environment was never watched")
	}

	select {
	case id := <-started:
		t.Fatalf("environment %d was watched twice across refreshes", id)
	case <-time.After(200 * time.Millisecond):
	}
}
