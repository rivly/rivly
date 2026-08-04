package server

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWaitBlocksUntilBackgroundWorkFinishes(t *testing.T) {
	srv := newTestServer(t)

	release := make(chan struct{})
	srv.Background(context.Background(), func(context.Context) { <-release })

	returned := make(chan error, 1)
	go func() { returned <- srv.Wait(context.Background()) }()

	select {
	case err := <-returned:
		t.Fatalf("Wait returned while background work was still running: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(release)

	select {
	case err := <-returned:
		if err != nil {
			t.Fatalf("Wait: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Wait did not return once the background work finished")
	}
}

func TestWaitGivesUpAtTheDeadline(t *testing.T) {
	srv := newTestServer(t)

	release := make(chan struct{})
	defer close(release)
	srv.Background(context.Background(), func(context.Context) { <-release })

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := srv.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Wait past the deadline: want DeadlineExceeded, got %v", err)
	}
}

func TestBackgroundLoopsStopWithTheContext(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{}
	seedEnvironment(t, srv)

	ctx, cancel := context.WithCancel(context.Background())
	for _, loop := range []func(context.Context){srv.RunPoller, srv.RunWatchers, srv.RunGitPoller} {
		srv.Background(ctx, loop)
	}

	cancel()

	waitCtx, waitCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer waitCancel()
	if err := srv.Wait(waitCtx); err != nil {
		t.Fatalf("background loops did not stop with the context: %v", err)
	}
}
