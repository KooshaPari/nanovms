//go:build !linux

// Non-Linux stubs for the landlock exported wrappers.
// On non-Linux platforms, Landlock is not available.

package sandbox

import "fmt"

// BuildLandlockRulesetDefault is a no-op on non-Linux platforms.
func BuildLandlockRulesetDefault() (int, error) {
	return -1, fmt.Errorf("landlock: not supported on this platform")
}

// LandlockRestrictSelf is a no-op on non-Linux platforms.
func LandlockRestrictSelf(rulesetFd int) error {
	return fmt.Errorf("landlock: not supported on this platform")
}
