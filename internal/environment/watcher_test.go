package environment

import (
	"context"
	"testing"
	"time"
)

func TestWatchersPickUpAnEnvironmentTheyMissedAtStartup(t *testing.T) {
	started := make(chan int64, 4)
	h := newWatcherHarness(t, fakeDocker{watchStarted: started}.factory())

	previous := watcherRefresh
	watcherRefresh = 20 * time.Millisecond
	t.Cleanup(func() { watcherRefresh = previous })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		if err := h.wait(stopCtx); err != nil {
			t.Errorf("watchers did not stop: %v", err)
		}
	})

	h.background(ctx, h.RunWatchers)

	select {
	case id := <-started:
		t.Fatalf("watched environment %d before any existed", id)
	case <-time.After(100 * time.Millisecond):
	}

	h.seedEnvironment(t)

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("an environment that appeared after startup was never watched")
	}
}

func TestWatchersDoNotStartTwiceForTheSameEnvironment(t *testing.T) {
	started := make(chan int64, 8)
	h := newWatcherHarness(t, fakeDocker{watchStarted: started}.factory())
	h.seedEnvironment(t)

	previous := watcherRefresh
	watcherRefresh = 10 * time.Millisecond
	t.Cleanup(func() { watcherRefresh = previous })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		if err := h.wait(stopCtx); err != nil {
			t.Errorf("watchers did not stop: %v", err)
		}
	})

	h.background(ctx, h.RunWatchers)

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
