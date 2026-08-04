package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHubPublishReceive(t *testing.T) {
	hub := NewHub()
	sub, unsubscribe := hub.Subscribe()
	defer unsubscribe()

	hub.Publish("environment.updated", map[string]int{"id": 7})

	select {
	case evt := <-sub.Events():
		if evt.Type != "environment.updated" {
			t.Fatalf("type: got %q", evt.Type)
		}
		var payload map[string]int
		if err := json.Unmarshal(evt.Data, &payload); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if payload["id"] != 7 {
			t.Fatalf("payload: got %+v", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestHubSlowSubscriberDoesNotBlock(t *testing.T) {
	hub := NewHub()
	_, unsubscribe := hub.Subscribe()
	defer unsubscribe()

	done := make(chan struct{})
	go func() {
		for range 1000 {
			hub.Publish("noise", 1)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("publish blocked on a slow subscriber")
	}
}

func TestHubMarksASlowSubscriberStale(t *testing.T) {
	hub := NewHub()
	sub, unsubscribe := hub.Subscribe()
	defer unsubscribe()

	for range subscriberBuffer {
		hub.Publish("noise", 1)
	}

	select {
	case <-sub.Stale():
		t.Fatal("a subscriber that kept up must not be marked stale")
	default:
	}
	if got := hub.Dropped(); got != 0 {
		t.Fatalf("dropped = %d before the buffer is full, want 0", got)
	}

	hub.Publish("overflow", 1)

	select {
	case <-sub.Stale():
	default:
		t.Fatal("a subscriber that missed an event must be told, or its view stays stale forever")
	}
	if got := hub.Dropped(); got != 1 {
		t.Fatalf("dropped = %d, want 1", got)
	}
}

func TestHubStaleIsReportedOnlyOnce(t *testing.T) {
	hub := NewHub()
	sub, unsubscribe := hub.Subscribe()
	defer unsubscribe()

	for range subscriberBuffer + 10 {
		hub.Publish("noise", 1)
	}

	select {
	case <-sub.Stale():
	default:
		t.Fatal("expected the subscriber to be stale")
	}
	if got := hub.Dropped(); got != 10 {
		t.Fatalf("dropped = %d, want 10", got)
	}
}

func TestHubUnsubscribeMarksTheSubscriptionStale(t *testing.T) {
	hub := NewHub()
	sub, unsubscribe := hub.Subscribe()
	unsubscribe()

	select {
	case <-sub.Stale():
	default:
		t.Fatal("unsubscribing must release a reader blocked on the subscription")
	}

	hub.Publish("after", 1)
	if got := hub.Dropped(); got != 0 {
		t.Fatalf("an unsubscribed reader must not count as a drop, got %d", got)
	}
}
