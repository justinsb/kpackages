package kmod

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func LoadModule(modulePath string, params string, flags int) error {
	f, err := os.Open(modulePath)
	if err != nil {
		return fmt.Errorf("error opening file %q: %w", modulePath, err)
	}
	defer f.Close()

	if err := unix.FinitModule(int(f.Fd()), params, flags); err != nil {
		return fmt.Errorf("error from finit(%q, %q, %v): %w", modulePath, params, flags, err)
	}
	return nil
}
