package main

import (
	"fmt"
	"os"

	"github.com/roman-kulish/locations-visualisation-example/cmd/server/app"
)

func main() {
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s", err)
		os.Exit(1)
	}
	os.Exit(0)
}
