package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	envProjectID                     = "PROJECT_ID"
	envCellEdgeLength                = "CELL_EDGE_LENGTH"
	envWindowDuration                = "WINDOW_DURATION"
	envSubscriptionID                = "SUBSCRIPTION_ID"
	envShutdownTimeout               = "SHUTDOWN_TIMEOUT"
	envReceiveRetryDelay             = "RECEIVE_RETRY_DELAY"
	envReceiveMaxExtension           = "RECEIVE_MAX_EXTENSION"
	envReceiveMaxExtensionPeriod     = "RECEIVE_MAX_EXTENSION_PERIOD"
	envReceiveMaxOutstandingMessages = "RECEIVE_MAX_OUTSTANDING_MESSAGES"
	envReceiveMaxOutstandingBytes    = "RECEIVE_MAX_OUTSTANDING_BYTES"
	envReceiveNumGoroutines          = "RECEIVE_NUM_GOROUTINES"
	envReceiveSynchronous            = "RECEIVE_SYNCHRONOUS"
	envServerAddr                    = "SERVER_ADDR"
	envServerReadTimeout             = "SERVER_READ_TIMEOUT"
	envServerReadHeaderTimeout       = "SERVER_READ_HEADER_TIMEOUT"
	envServerWriteTimeout            = "SERVER_WRITE_TIMEOUT"
	envServerIdleTimeout             = "SERVER_IDLE_TIMEOUT"
)

var envVars = []string{
	envProjectID,
	envCellEdgeLength,
	envWindowDuration,
	envSubscriptionID,
	envShutdownTimeout,
	envReceiveRetryDelay,
	envReceiveMaxExtension,
	envReceiveMaxExtensionPeriod,
	envReceiveMaxOutstandingMessages,
	envReceiveMaxOutstandingBytes,
	envReceiveNumGoroutines,
	envReceiveSynchronous,
	envServerAddr,
	envServerReadTimeout,
	envServerReadHeaderTimeout,
	envServerWriteTimeout,
	envServerIdleTimeout,
}

// NewFromEnv returns configuration created from environment variables.
func NewFromEnv() (*Config, error) {
	cfg := newConfig()

	for _, key := range envVars {
		val := os.Getenv(key)
		if val == "" {
			continue
		}

		var err error
		switch key {
		case envProjectID:
			cfg.ProjectID = val
		case envCellEdgeLength:
			cfg.CellEdgeLength, err = strconv.ParseFloat(val, 64)
		case envWindowDuration:
			cfg.WindowDuration, err = time.ParseDuration(val)
		case envSubscriptionID:
			cfg.Subscription.ID = val
		case envShutdownTimeout:
			cfg.ShutdownTimeout, err = time.ParseDuration(val)
		case envReceiveRetryDelay:
			cfg.Subscription.RetryDelay, err = time.ParseDuration(val)
		case envReceiveMaxExtension:
			cfg.Subscription.Settings.MaxExtension, err = time.ParseDuration(val)
		case envReceiveMaxExtensionPeriod:
			cfg.Subscription.Settings.MaxExtensionPeriod, err = time.ParseDuration(val)
		case envReceiveMaxOutstandingMessages:
			cfg.Subscription.Settings.MaxOutstandingMessages, err = strconv.Atoi(val)
		case envReceiveMaxOutstandingBytes:
			cfg.Subscription.Settings.MaxOutstandingBytes, err = strconv.Atoi(val)
		case envReceiveNumGoroutines:
			cfg.Subscription.Settings.NumGoroutines, err = strconv.Atoi(val)
		case envReceiveSynchronous:
			cfg.Subscription.Settings.Synchronous, err = strconv.ParseBool(val)
		case envServerAddr:
			cfg.Server.Addr = val
		case envServerReadTimeout:
			cfg.Server.ReadTimeout, err = time.ParseDuration(val)
		case envServerReadHeaderTimeout:
			cfg.Server.ReadHeaderTimeout, err = time.ParseDuration(val)
		case envServerWriteTimeout:
			cfg.Server.WriteTimeout, err = time.ParseDuration(val)
		case envServerIdleTimeout:
			cfg.Server.IdleTimeout, err = time.ParseDuration(val)
		}
		if err != nil {
			return nil, fmt.Errorf(`config: error parsing value of "%s": %w`, key, err)
		}
	}
	return cfg, nil
}
