package config

import (
	"time"

	"cloud.google.com/go/pubsub"
)

const defaultPort = ":8080"

// Subscription contains configuration for PubSub Subscription.
type Subscription struct {
	// ID is a subscription ID to pull events from.
	ID string

	// RetryDelay is a time between PubSub receive retries.
	RetryDelay time.Duration

	// Settings contains subscription receive configuration.
	Settings pubsub.ReceiveSettings
}

type Server struct {
	// Addr specifies the address for the server to listen on.
	Addr string

	// ReadTimeout is the maximum duration for reading the entire
	// request, including the body.
	ReadTimeout time.Duration

	// ReadHeaderTimeout is the amount of time allowed to read
	// request headers.
	ReadHeaderTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out
	// writes of the response.
	WriteTimeout time.Duration

	// IdleTimeout is the maximum amount of time to wait for the
	// next request when keep-alives are enabled. If IdleTimeout
	// is zero, the value of ReadTimeout is used. If both are
	// zero, there is no timeout.
	IdleTimeout time.Duration
}

// Config provides configuration to the application.
type Config struct {
	// ProjectID is a Google project ID.
	ProjectID string

	// Cell edge length in meters, used to find approximate cell
	// level to use.
	CellEdgeLength float64

	// WindowDuration is how long data should be accumulated before
	// sending result out.
	WindowDuration time.Duration

	// ShutdownTimeout is a server graceful shutdown timeout.
	ShutdownTimeout time.Duration

	Server
	Subscription
}

// newConfig returns default configuration. Note that this configuration
// is not usable.
func newConfig() *Config {
	return &Config{
		Server: Server{
			Addr: defaultPort,
		},
	}
}
