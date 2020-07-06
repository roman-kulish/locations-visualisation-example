package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/pubsub"

	"github.com/roman-kulish/locations-visualisation-example/cmd/server/app/accumulator"
	"github.com/roman-kulish/locations-visualisation-example/cmd/server/app/brocker"
	"github.com/roman-kulish/locations-visualisation-example/cmd/server/app/config"

	htServer "github.com/roman-kulish/locations-visualisation-example/cmd/server/app/http"
	psServer "github.com/roman-kulish/locations-visualisation-example/cmd/server/app/pubsub"
)

func startPubSub(ctx context.Context, cfg *config.Subscription, cl *pubsub.Client, acc *accumulator.Accumulator, shutdownTimeout time.Duration, done chan<- error) {
	sub := cl.Subscription(cfg.ID)
	sub.ReceiveSettings = cfg.Settings

	srv := psServer.New(acc, sub, cfg.RetryDelay)

	go func() {
		<-ctx.Done()

		cctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		done <- srv.Shutdown(cctx)
	}()

	done <- srv.Start()
}

func startServer(ctx context.Context, cfg *config.Server, br *brocker.Broker, shutdownTimeout time.Duration, done chan<- error) {
	srv, err := htServer.New(ctx, cfg, br)
	if err != nil {
		done <- err
		return
	}

	go func() {
		<-ctx.Done()

		cctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		done <- srv.Shutdown(cctx)
	}()

	done <- srv.ListenAndServe()
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

	accBufSize := pubsub.DefaultReceiveSettings.MaxOutstandingMessages
	if n := cfg.Subscription.Settings.MaxOutstandingMessages; n > 0 {
		accBufSize = n
	}

	br := brocker.New()
	acc, err := accumulator.New(cfg.CellEdgeLength, cfg.WindowDuration, accBufSize, resultHandler(br))
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
	ctx, cancel := context.WithCancel(context.Background())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		cancel()
	}()

	go startPubSub(ctx, &cfg.Subscription, cl, acc, cfg.ShutdownTimeout, done)
	go startServer(ctx, &cfg.Server, br, cfg.ShutdownTimeout, done)

	var cancelled bool
	for i := 0; i < cap(done); i++ {
		if e := <-done; e != nil && err == nil {
			err = e
		}
		if !cancelled {
			cancel()
			cancelled = true
		}
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
