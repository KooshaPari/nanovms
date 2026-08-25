//go:build linux

// Exported wrappers for the landlock kernel-syscall helpers.
// These are called from pkg/tier/landlock.go to apply real Landlock
// filesystem restrictions to the calling thread.

package sandbox

import "golang.org/x/sys/unix"

// BuildLandlockRulesetDefault constructs a default read-only Landlock ruleset
// that allows reading everything but denies writes (except /tmp which is
// read-write). Returns the ruleset fd (caller must close via LandlockRestrictSelf).
func BuildLandlockRulesetDefault() (int, error) {
	readOnlyPaths := []string{"/"}
	readWritePaths := []string{"/tmp"}
	return buildLandlockRulesetStub(readOnlyPaths, readWritePaths)
}

// LandlockRestrictSelf applies the given Landlock ruleset fd to the calling
// thread via PR_SET_NO_NEW_PRIVS + LANDLOCK_RESTRICT_SELF, then closes the fd.
func LandlockRestrictSelf(rulesetFd int) error {
	defer unix.Close(rulesetFd)
	return landlockRestrictSelfStub(rulesetFd)
}
