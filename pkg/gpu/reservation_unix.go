// SPDX-License-Identifier: MIT OR Apache-2.0
//go:build !windows

package gpu

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func tryLockReservationFile(file *os.File) (bool, error) {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return false, nil
	}
	return err == nil, err
}

func unlockReservationFile(file *os.File) {
	_ = unix.Flock(int(file.Fd()), unix.LOCK_UN)
	_ = file.Close()
}

func replaceFile(source, target string) error {
	return os.Rename(source, target)
}
