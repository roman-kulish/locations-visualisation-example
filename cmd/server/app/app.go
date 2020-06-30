package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"cloud.google.com/go/pubsub"

	"github.com/roman-kulish/locations-visualisation-example/cmd/server/app/accumulator"
	"github.com/roman-kulish/locations-visualisation-example/cmd/server/app/brocker"
	"github.com/roman-kulish/locations-visualisation-example/cmd/server/app/config"

	htServer "github.com/roman-kulish/locations-visualisation-example/cmd/server/app/http"
	psServer "github.com/roman-kulish/locations-visualisation-example/cmd/server/app/pubsub"
)

type shutdownNotifier struct {
	mu         sync.Mutex
	inShutdown bool
	done       chan struct{}
}

func newShutdownNotifier() *shutdownNotifier {
	return &shutdownNotifier{
		done: make(chan struct{}),
	}
}

func (sn *shutdownNotifier) Done() <-chan struct{} {
	return sn.done
}

func (sn *shutdownNotifier) Notify() {
	sn.mu.Lock()
	if sn.inShutdown {
		sn.mu.Unlock()
		return
	}

	sn.inShutdown = true
	close(sn.done)
	sn.mu.Unlock()
}

func startPubSub(cfg *config.Subscription,
	cl *pubsub.Client, acc *accumulator.Accumulator,
	shutdownTimeout time.Duration,
	done chan<- error,
	stop *shutdownNotifier) error {

	sub := cl.Subscription(cfg.ID)
	sub.ReceiveSettings = cfg.Settings

	srv := psServer.New(acc, sub, cfg.RetryDelay)

	go func() {
		<-stop.Done()

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		done <- srv.Shutdown(ctx)
	}()

	return srv.Start()
}

func startServer(cfg *config.Server,
	br *brocker.Broker,
	shutdownTimeout time.Duration,
	done chan<- error,
	stop *shutdownNotifier) error {

	srv, err := htServer.New(cfg, br)
	if err != nil {
		return err
	}

	go func() {
		<-stop.Done()

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		done <- srv.Shutdown(ctx)
	}()

	return srv.ListenAndServe()
}

// Run is the second "main", which initialises and wires services and starts
// the application. Services which must be finalised on termination can
// register their finalise methods via defer. Run function does not panic or
// terminates via os.Exit() and so it guarantees graceful services shutdown.
func Run() (err error) {
	defer errorHandler(&err)

	cfg, err := config.NewFromEnv()
	if err != nil {
		return err
	}

	accBufSIze := pubsub.DefaultReceiveSettings.MaxOutstandingMessages
	if n := cfg.Subscription.Settings.MaxOutstandingMessages; n > 0 {
		accBufSIze = n
	}

	br := brocker.New()
	acc, err := accumulator.New(
		cfg.CellEdgeLength,
		cfg.WindowDuration,
		accBufSIze,
		resultHandler(br),
	)
	if err != nil {
		return err
	}
	defer acc.Stop()

	cl, err := pubsub.NewClient(context.Background(), cfg.ProjectID)
	if err != nil {
		return err
	}
	defer cl.Close()

	done := make(chan error, 4)
	stop := newShutdownNotifier()

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

		<-quit
		stop.Notify()
	}()
	go func() {
		done <- startPubSub(&cfg.Subscription, cl, acc, cfg.ShutdownTimeout, done, stop)
	}()
	go func() {
		done <- startServer(&cfg.Server, br, cfg.ShutdownTimeout, done, stop)
	}()

	for i := 0; i < cap(done); i++ {
		if e := <-done; e != nil {
			err = e
		}
		stop.Notify()
	}

	close(done)
	return err
}

func resultHandler(b *brocker.Broker) accumulator.OnResultFunc {
	return func(r accumulator.Result) {
		b.Send(r)
	}
}

func errorHandler(err *error) {
	if r := recover(); r != nil {
		switch v := r.(type) {
		case error:
			*err = v
		default:
			*err = fmt.Errorf("panic: %v", v)
		}
	}
}
