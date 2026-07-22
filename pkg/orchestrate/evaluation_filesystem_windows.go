// SPDX-License-Identifier: MIT OR Apache-2.0
//go:build windows

package orchestrate

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

func isUnsafeOutputPathEntry(info os.FileInfo) bool {
	if info.Mode()&os.ModeSymlink != 0 {
		return true
	}
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	return ok && data.FileAttributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0
}

func outputRootAvailableSpace(path string) (uint64, error) {
	pathUTF16, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var available uint64
	if err := windows.GetDiskFreeSpaceEx(pathUTF16, &available, nil, nil); err != nil {
		return 0, err
	}
	return available, nil
}
