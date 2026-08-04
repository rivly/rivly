package stack

import (
	"context"
	"testing"
)

func TestValidAction(t *testing.T) {
	for _, action := range []string{"start", "stop", "restart", "remove"} {
		if !ValidAction(action) {
			t.Errorf("ValidAction(%q) = false, want true", action)
		}
	}
	for _, action := range []string{"", "pause", "kill", "unpause", "REMOVE"} {
		if ValidAction(action) {
			t.Errorf("ValidAction(%q) = true, want false", action)
		}
	}
}

func TestActRemovesAManagedStackAndForgetsIt(t *testing.T) {
	removed := make(chan string, 1)
	h := newHarness(t, fakeDocker{}, fakeCompose{removed: removed})
	h.seed(t, "app", fakeRemote(t, headA), headA, 15, 1)

	results := h.Act(context.Background(), h.env(t), "remove", []string{"app"})

	if len(results) != 1 || !results[0].OK {
		t.Fatalf("remove: got %+v", results)
	}
	select {
	case project := <-removed:
		if project != "app" {
			t.Errorf("compose tore down %q, want %q", project, "app")
		}
	default:
		t.Fatal("compose was never asked to tear the stack down")
	}

	if _, err := h.Get(context.Background(), h.env(t), "app"); kindOf(t, err) != KindNotFound {
		t.Error("a removed stack must no longer be known to Rivly")
	}
}

func TestActKeepsTheRecordWhenTeardownFails(t *testing.T) {
	h := newHarness(t, fakeDocker{}, fakeCompose{err: errComposeFailed})
	h.seed(t, "app", fakeRemote(t, headA), headA, 15, 1)

	results := h.Act(context.Background(), h.env(t), "remove", []string{"app"})
	if len(results) != 1 || results[0].OK {
		t.Fatalf("a failed teardown must be reported: %+v", results)
	}

	if _, err := h.Get(context.Background(), h.env(t), "app"); err != nil {
		t.Error("a failed teardown must not forget the stack, or it becomes unmanageable")
	}
}

func TestActRefusesToRemoveAStackBeingUpdated(t *testing.T) {
	h := newHarness(t, fakeDocker{}, fakeCompose{})
	record := h.seed(t, "app", fakeRemote(t, headA), headA, 15, 1)

	if !h.Acquire(record.ID) {
		t.Fatal("Acquire must succeed on a free stack")
	}
	defer h.Release(record.ID)

	results := h.Act(context.Background(), h.env(t), "remove", []string{"app"})
	if len(results) != 1 || results[0].OK {
		t.Fatalf("removing a stack under update must fail: %+v", results)
	}
	if results[0].Error != "an update is running" {
		t.Errorf("error = %q", results[0].Error)
	}
}

func TestActOnAnUnmanagedStackGoesToDocker(t *testing.T) {
	removed := make(chan string, 1)
	h := newHarness(t, fakeDocker{}, fakeCompose{removed: removed})

	results := h.Act(context.Background(), h.env(t), "remove", []string{"external"})

	if len(results) != 1 || !results[0].OK {
		t.Fatalf("remove: got %+v", results)
	}
	select {
	case project := <-removed:
		t.Fatalf("compose must not be used on a stack Rivly never deployed, got %q", project)
	default:
	}
}

func TestActReportsPerStackResults(t *testing.T) {
	h := newHarness(t, fakeDocker{actionErr: errComposeFailed}, fakeCompose{})

	results := h.Act(context.Background(), h.env(t), "restart", []string{"a", "b", "c"})
	if len(results) != 3 {
		t.Fatalf("want one result per stack, got %d", len(results))
	}
	for i, r := range results {
		if r.OK {
			t.Errorf("result %d: want a failure, got %+v", i, r)
		}
		if r.Name != []string{"a", "b", "c"}[i] {
			t.Errorf("result %d is out of order: %+v", i, r)
		}
	}
}
