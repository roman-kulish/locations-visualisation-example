package brocker

import (
	"sync"

	"github.com/roman-kulish/locations-visualisation-example/cmd/server/app/accumulator"
)

// Subscription is a broker subscription, which provides
// a channel to read events from.
type Subscription struct {
	br  *Broker
	idx uint64
	out chan accumulator.Result
}

// Close closes this subscription and unsubscribes subscriber from the broker.
func (s Subscription) Close() {
	s.br.Unsubscribe(s)
}

// Out returns a channel to read events from.
func (s Subscription) Out() <-chan accumulator.Result {
	return s.out
}

// Broker is a message broker.
type Broker struct {
	mu   sync.RWMutex
	idx  uint64
	subs map[uint64]chan accumulator.Result
}

// New creates and returns a new Broker instance.
func New() *Broker {
	return &Broker{
		subs: make(map[uint64]chan accumulator.Result),
	}
}

// Send sends accumulator result to subscribers.
func (b *Broker) Send(r accumulator.Result) {
	b.mu.RLock()
	for id := range b.subs {
		go func(ch chan<- accumulator.Result, r accumulator.Result) {
			ch <- r
		}(b.subs[id], r)
	}
	b.mu.RUnlock()
}

// Subscribe creates and returns a new Subscription instance.
// This instance is ready to read events from.
func (b *Broker) Subscribe() Subscription {
	b.mu.Lock()
	s := Subscription{
		br:  b,
		idx: b.idx,
		out: make(chan accumulator.Result, 1),
	}
	b.subs[s.idx] = s.out
	b.idx++
	b.mu.Unlock()
	return s
}

// Unsubscribe removes subscription from the broker.
func (b *Broker) Unsubscribe(s Subscription) {
	b.mu.Lock()
	ch, ok := b.subs[s.idx]
	if !ok {
		b.mu.Unlock()
		return
	}

	delete(b.subs, s.idx)
	close(ch)
	b.mu.Unlock()
}
