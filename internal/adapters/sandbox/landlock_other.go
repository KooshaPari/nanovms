//go:build !linux

package sandbox

import "fmt"

// kernelSupportsLandlock returns false on non-Linux platforms.
// Landlock is a Linux-specific kernel feature (>= 5.13).
func kernelSupportsLandlockWrapper() bool { return false }

// buildLandlockRuleset is a no-op stub on non-Linux platforms.
// On Linux, see landlock_linux.go.
func buildLandlockRulesetStub(readOnlyPaths, readWritePaths []string) (int, error) {
	return -1, fmt.Errorf("landlock: not supported on this platform (Linux only)")
}

// landlockRestrictSelfStub is a no-op on non-Linux platforms.
func landlockRestrictSelfStub(fd int) error {
	return fmt.Errorf("landlock: not supported on this platform (Linux only)")
}
