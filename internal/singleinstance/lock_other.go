//go:build !windows

package singleinstance

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

var lockFile *os.File

func Acquire() (bool, error) {
	// flock-style via exclusive create of lock file in temp — use data dir via env later.
	// For non-Windows, use a lock file in OS temp.
	path := filepath.Join(os.TempDir(), "pakegui.lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return false, err
	}
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		_ = f.Close()
		return false, nil
	}
	lockFile = f
	_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
	return true, nil
}

func Release() {
	if lockFile == nil {
		return
	}
	_ = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
	_ = lockFile.Close()
	lockFile = nil
}

func ActivateExisting() bool {
	return false
}
