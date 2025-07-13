//go:build windows

package logma

import (
	"os"

	"golang.org/x/sys/windows"
)

func isatty(f *os.File) bool {
	var mode uint32
	err := windows.GetConsoleMode(windows.Handle(f.Fd()), &mode)
	return err == nil
}
