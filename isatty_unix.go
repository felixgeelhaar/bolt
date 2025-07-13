//go:build !windows

package logma

import (
	"os"

	"golang.org/x/term"
)

func isatty(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}
