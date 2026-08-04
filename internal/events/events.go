package events

import (
	"encoding/json"
	"sync"
	"sync/atomic"
)

const subscriberBuffer = 16

type Event struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type Subscription struct {
	events chan Event
	stale  chan struct{}
	once   sync.Once
}

func (s *Subscription) Events() <-chan Event {
	return s.events
}

func (s *Subscription) Stale() <-chan struct{} {
	return s.stale
}

func (s *Subscription) markStale() {
	s.once.Do(func() { close(s.stale) })
}

type Hub struct {
	mu      sync.RWMutex
	subs    map[*Subscription]struct{}
	dropped atomic.Uint64
}

func NewHub() *Hub {
	return &Hub{subs: make(map[*Subscription]struct{})}
}

func (h *Hub) Subscribe() (*Subscription, func()) {
	sub := &Subscription{
		events: make(chan Event, subscriberBuffer),
		stale:  make(chan struct{}),
	}

	h.mu.Lock()
	h.subs[sub] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.subs, sub)
			h.mu.Unlock()
			sub.markStale()
		})
	}
	return sub, unsubscribe
}

func (h *Hub) Publish(eventType string, data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	evt := Event{Type: eventType, Data: raw}

	h.mu.RLock()
	defer h.mu.RUnlock()
	for sub := range h.subs {
		select {
		case sub.events <- evt:
		default:
			h.dropped.Add(1)
			sub.markStale()
		}
	}
}

func (h *Hub) Dropped() uint64 {
	return h.dropped.Load()
}
