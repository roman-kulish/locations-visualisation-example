package pubsub

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"cloud.google.com/go/pubsub"

	"github.com/roman-kulish/locations-visualisation-example/cmd/server/app/accumulator"
)

const (
	defaultRetryDelay = 5 * time.Second
	pollRetry         = 200 * time.Millisecond
)

var ErrServerClosed = errors.New("server: server closed")

type atomicBool int32

func (b *atomicBool) isSet() bool {
	return atomic.LoadInt32((*int32)(b)) != 0
}

func (b *atomicBool) setTrue() {
	atomic.StoreInt32((*int32)(b), 1)
}

type event struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"long"`
}

// Server implements a PubSub messages server.
type Server struct {
	sub        *pubsub.Subscription
	acc        *accumulator.Accumulator
	delay      time.Duration
	cancel     context.CancelFunc
	inShutdown atomicBool
	wg         sync.WaitGroup
}

// New creates and returns a new PubSub server instance.
func New(acc *accumulator.Accumulator, sub *pubsub.Subscription, retryDelay time.Duration) *Server {
	delay := defaultRetryDelay
	if retryDelay > 0 {
		delay = retryDelay
	}
	return &Server{
		sub:   sub,
		acc:   acc,
		delay: delay,
	}
}

// Start accepting PubSub messages.
func (s *Server) Start() error {
	if s.inShutdown.isSet() {
		return ErrServerClosed
	}

	var ctx context.Context
	ctx, s.cancel = context.WithCancel(context.Background())

	return s.receiveWithRetry(ctx)
}

// Shutdown the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.inShutdown.setTrue()
	s.cancel()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		done <- struct{}{}
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// receiveWithRetry blocks until context is cancelled by calling Shutdown().
// If subscription Receive() exits or returns error, this method will wait
// some time and retry.
func (s *Server) receiveWithRetry(ctx context.Context) error {
	err := s.startReceiver(ctx)
	for {
		// If ctx is done, Receive returns nil after all of the outstanding calls
		// to f have returned and all messages have been acknowledged or have expired.
		//
		// https://pkg.go.dev/cloud.google.com/go/pubsub?tab=doc#Subscription.Receive
		if e := ctx.Err(); e != nil {
			// context was cancelled, return immediately.
			return e
		}

		// If the service returns a non-retryable error, Receive returns that
		// error after all of the outstanding calls to f have returned.
		//
		// https://pkg.go.dev/cloud.google.com/go/pubsub?tab=doc#Subscription.Receive
		if err = sleep(ctx, s.delay); err != nil {
			// context was cancelled, return immediately.
			return err
		}

		// Retry receiving from PubSub.
		err = s.startReceiver(ctx)
	}
}

// startReceiver starts subscription Receive().
func (s *Server) startReceiver(ctx context.Context) error {
	defer s.wg.Done()

	s.wg.Add(1)

	// Reason for this: s.sub.Receive() may fail and exit, leaving goroutines
	// running. Using child context and cancelling it on startReceiver() exit
	// guarantees that these goroutines will receive cancellation signal.
	//
	// If server is going to shutdown in a proper way, then parent context will
	// close cctx Done channel.
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()

	return s.sub.Receive(cctx, func(ctx context.Context, msg *pubsub.Message) {
		defer msg.Ack()

		var ev event
		if err := json.Unmarshal(msg.Data, &ev); err != nil {
			return
		}
		if ev.Lat == 0 && ev.Lng == 0 {
			return
		}

		s.acc.Add(ev.Lat, ev.Lng)
	})
}

// sleep is similar to time.Sleep, but it can be interrupted by ctx.Done() closing.
// If interrupted, sleep returns ctx.Err().
func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
