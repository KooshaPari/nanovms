// SPDX-License-Identifier: MIT OR Apache-2.0
// Package tier — helpers.go contains small shared helpers used by the
// adapter files.
package tier

import "os"

// probeOverride reads an environment variable used to override probe
// results during testing. Returns "" if the variable is unset, "0" if it
// is explicitly set to "0" (force probe to succeed), or "1" (force probe
// to fail). Any other value is treated as unset.
func probeOverride(name string) string {
	v, ok := os.LookupEnv(name)
	if !ok {
		return ""
	}
	return v
}
