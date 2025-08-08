package main

import (
	"os"

	"github.com/felixgeelhaar/bolt"
)

func main() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().Msg("Test successful - Bolt v1.2.0 is working!")
}
