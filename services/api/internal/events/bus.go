// Package events provides an in-process event bus that powers all API
// triggers (emails, SMS, analytics, scheduler jobs). Events are dispatched
// asynchronously to registered handlers by a bounded worker pool so a slow
// handler (e.g. an SMTP send) never blocks the HTTP response.
package events

import (
	"context"
	"log"
	"sync"
)

// Event is a domain event fired by services. Payload is JSON-serializable.
type Event struct {
	Name    string
	Payload interface{}
}

// Handler processes a single event. It must be safe for concurrent use.
type Handler func(ctx context.Context, e Event) error

// Bus dispatches events to handlers.
type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
	ch       chan Event
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	closed   bool
}

// New creates a Bus with a bounded queue and worker pool.
func New(workers int, queueSize int) *Bus {
	ctx, cancel := context.WithCancel(context.Background())
	b := &Bus{
		handlers: make(map[string][]Handler),
		ch:       make(chan Event, queueSize),
		ctx:      ctx,
		cancel:   cancel,
	}
	for i := 0; i < workers; i++ {
		b.wg.Add(1)
		go b.worker()
	}
	return b
}

func (b *Bus) worker() {
	defer b.wg.Done()
	for {
		select {
		case <-b.ctx.Done():
			return
		case e, ok := <-b.ch:
			if !ok {
				return
			}
			b.dispatch(e)
		}
	}
}

func (b *Bus) dispatch(e Event) {
	b.mu.RLock()
	handlers := b.handlers[e.Name]
	b.mu.RUnlock()
	for _, h := range handlers {
		if err := h(b.ctx, e); err != nil {
			log.Printf("events: handler for %q failed: %v", e.Name, err)
		}
	}
}

// On registers a handler for an event name.
func (b *Bus) On(name string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[name] = append(b.handlers[name], h)
}

// Publish queues an event. It never blocks on a full queue; if the queue is
// full the event is dropped and logged, since handlers re-read state from the
// database anyway.
func (b *Bus) Publish(name string, payload interface{}) {
	if b.closed {
		return
	}
	select {
	case b.ch <- Event{Name: name, Payload: payload}:
	default:
		log.Printf("events: queue full, dropping %q", name)
	}
}

// Close drains the queue and stops workers.
func (b *Bus) Close() {
	if b.closed {
		return
	}
	b.closed = true
	b.cancel()
	close(b.ch)
	b.wg.Wait()
}
