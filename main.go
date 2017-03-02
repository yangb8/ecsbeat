package main

import (
	"os"

	"github.com/elastic/beats/libbeat/beat"

	"github.com/yangb8/ecsbeat/beater"
)

func main() {
	err := beat.Run("ecsbeat", "", beater.New)
	if err != nil {
		os.Exit(1)
	}
}
