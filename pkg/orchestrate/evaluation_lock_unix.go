// SPDX-License-Identifier: MIT OR Apache-2.0
//go:build !windows

package orchestrate

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func tryLockOutputFile(file *os.File) (bool, error) {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return false, nil
	}
	return err == nil, err
}

func unlockOutputFile(file *os.File) {
	_ = unix.Flock(int(file.Fd()), unix.LOCK_UN)
	_ = file.Close()
}
