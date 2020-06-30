package http

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/roman-kulish/locations-visualisation-example/cmd/server/app/brocker"
	"github.com/roman-kulish/locations-visualisation-example/cmd/server/app/config"
)

func newStreamHandler(br *brocker.Broker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming is not supported", http.StatusNotImplemented)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		sub := br.Subscribe()
		for {
			select {
			case ev := <-sub.Out():
				fmt.Fprintf(w, "data: [%0.5f, %0.5f, %0.2f]\n\n", ev.Lat, ev.Lng, float64(ev.Count)/float64(ev.Max))
				f.Flush()

			case <-r.Context().Done():
				sub.Close()
				return
			}
		}
	}
}

// New creates and returns a new configured instance of HTTP server.
func New(cfg *config.Server, br *brocker.Broker) (*http.Server, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	dir = filepath.Join(dir, "public")
	mux := http.NewServeMux()
	mux.HandleFunc("/stream", newStreamHandler(br))
	mux.Handle("/", http.FileServer(http.Dir(dir)))

	srv := http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}
	return &srv, nil
}
