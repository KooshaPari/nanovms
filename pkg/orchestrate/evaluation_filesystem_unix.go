// SPDX-License-Identifier: MIT OR Apache-2.0
//go:build !windows

package orchestrate

import (
	"os"

	"golang.org/x/sys/unix"
)

func isUnsafeOutputPathEntry(info os.FileInfo) bool {
	return info.Mode()&os.ModeSymlink != 0
}

func outputRootAvailableSpace(path string) (uint64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return stat.Bavail * uint64(stat.Bsize), nil
}
