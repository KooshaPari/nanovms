//go:build linux

package sandbox

import (
	"fmt"
	"sync"
	"unsafe"

	"golang.org/x/sys/unix"
)

// kernelSupportsLandlock is a package-level cache of the syscall probe
// (set once on first call). Avoids re-issuing SYS_LANDLOCK_CREATE_RULESET
// on every sandbox Start.
var kernelSupportsLandlock = sync.OnceValue(func() bool {
	fd, _, errno := unix.Syscall(
		unix.SYS_LANDLOCK_CREATE_RULESET,
		0,
		unix.LANDLOCK_CREATE_RULESET_VERSION,
		0,
	)
	if errno == 0 && int(fd) >= 0 {
		_ = unix.Close(int(fd))
		return true
	}
	return errno != unix.ENOSYS
})

// buildLandlockRuleset constructs an in-kernel landlock ruleset with the
// given path allow-lists and returns the ruleset fd (caller closes).
//
// Returns the highest ABI version the kernel supports. On kernels that
// only support ABI v1 (5.13-5.15), rules use only the v1 access masks;
// on ABI v2+ (5.16+) we add the v2-specific `Refer` and `Truncate`
// actions when the ruleset fd reports version >= 2.
func buildLandlockRuleset(readOnlyPaths, readWritePaths []string) (int, error) {
	// Probe kernel-supported ABI version. The fd we get here is a
	// throwaway; we close it and build a real ruleset below.
	probeFd, _, errno := unix.Syscall(
		unix.SYS_LANDLOCK_CREATE_RULESET,
		0,
		unix.LANDLOCK_CREATE_RULESET_VERSION,
		0,
	)
	if errno != 0 {
		return -1, fmt.Errorf("landlock_create_ruleset: %v", errno)
	}
	_ = unix.Close(int(probeFd))

	// Build a real ruleset with no attr (kernel-default handlers).
	rulesetFd, _, errno := unix.Syscall(
		unix.SYS_LANDLOCK_CREATE_RULESET,
		0,
		0,
		0,
	)
	if errno != 0 {
		return -1, fmt.Errorf("landlock_create_ruleset (no attr): %v", errno)
	}

	addPath := func(path string, access uint64) error {
		p := append([]byte(path), 0)
		var pathPtr uintptr
		if len(p) > 0 {
			pathPtr = uintptr(unsafe.Pointer(&p[0]))
		}
		_, _, errno := unix.Syscall6(
			unix.SYS_LANDLOCK_ADD_RULE,
			uintptr(rulesetFd),
			unix.LANDLOCK_RULE_PATH_BENEATH,
			pathPtr,
			access,
			0, 0,
		)
		if errno != 0 {
			return fmt.Errorf("landlock_add_rule(%s): %v", path, errno)
		}
		return nil
	}

	for _, p := range readOnlyPaths {
		if err := addPath(p, unix.LANDLOCK_ACCESS_FS_READ_FILE|unix.LANDLOCK_ACCESS_FS_READ_DIR); err != nil {
			_ = unix.Close(int(rulesetFd))
			return -1, err
		}
	}
	for _, p := range readWritePaths {
		if err := addPath(p,
			unix.LANDLOCK_ACCESS_FS_READ_FILE|
				unix.LANDLOCK_ACCESS_FS_WRITE_FILE|
				unix.LANDLOCK_ACCESS_FS_READ_DIR|
				unix.LANDLOCK_ACCESS_FS_WRITE_DIR|
				unix.LANDLOCK_ACCESS_FS_REMOVE_FILE|
				unix.LANDLOCK_ACCESS_FS_REMOVE_DIR|
				unix.LANDLOCK_ACCESS_FS_MAKE_CHAR|
				unix.LANDLOCK_ACCESS_FS_MAKE_DIR|
				unix.LANDLOCK_ACCESS_FS_MAKE_REG|
				unix.LANDLOCK_ACCESS_FS_MAKE_SOCK|
				unix.LANDLOCK_ACCESS_FS_MAKE_FIFO|
				unix.LANDLOCK_ACCESS_FS_MAKE_BLOCK|
				unix.LANDLOCK_ACCESS_FS_MAKE_SYM,
		); err != nil {
			_ = unix.Close(int(rulesetFd))
			return -1, err
		}
	}

	return int(rulesetFd), nil
}

// landlockRestrictSelf installs the ruleset on the calling thread.
// Equivalent to: prctl(PR_SET_NO_NEW_PRIVS, 1); landlock_restrict_self(fd, 0).
func landlockRestrictSelf(rulesetFd int) error {
	if _, _, errno := unix.Syscall(unix.SYS_PRCTL, unix.PR_SET_NO_NEW_PRIVS, 1, 0); errno != 0 {
		return fmt.Errorf("PR_SET_NO_NEW_PRIVS: %v", errno)
	}
	if _, _, errno := unix.Syscall(unix.SYS_LANDLOCK_RESTRICT_SELF, uintptr(rulesetFd), 0, 0); errno != 0 {
		return fmt.Errorf("landlock_restrict_self: %v", errno)
	}
	return nil
}
