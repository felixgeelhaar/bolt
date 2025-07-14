//go:build tools

package main

// This file imports packages that are used when running tests.
// This allows us to version pin the tools used for benchmarking without
// adding them as dependencies to the main module.

import (
	_ "github.com/rs/zerolog"
	_ "go.uber.org/zap"
	_ "golang.org/x/exp/slog"
)