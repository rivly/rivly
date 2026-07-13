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
	case evt := <-sub:
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

	// The subscriber never reads; a full buffer must not block the publisher.
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

func TestHubUnsubscribe(t *testing.T) {
	hub := NewHub()
	sub, unsubscribe := hub.Subscribe()
	unsubscribe()

	if _, ok := <-sub; ok {
		t.Fatal("expected channel to be closed after unsubscribe")
	}
}
