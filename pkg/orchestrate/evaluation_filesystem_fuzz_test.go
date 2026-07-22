// SPDX-License-Identifier: MIT OR Apache-2.0
package orchestrate

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func FuzzValidateManagedAbsolutePath(f *testing.F) {
	f.Add(`C:\harbor\jobs`)
	f.Add(`/var/harbor/jobs`)
	f.Add(``)
	f.Add(`..`)
	f.Add(`C:\`)
	f.Add(`/`)
	f.Add(`\\server\share`)
	f.Add(`C:\harbor\..\etc`)
	f.Fuzz(func(t *testing.T, path string) {
		err := validateManagedAbsolutePath(path, "output_root", CodeInvalidOutputRoot)
		if err == nil {
			if path == "" || strings.TrimSpace(path) == "" || strings.ContainsRune(path, '\x00') {
				t.Fatalf("accepted empty/null path %q", path)
			}
			if !filepath.IsAbs(path) || filepath.Clean(path) != path || filepath.Dir(path) == path {
				t.Fatalf("accepted unsafe absolute path %q", path)
			}
			return
		}
		var evaluationErr *EvaluationError
		if !errors.As(err, &evaluationErr) || evaluationErr.Code != CodeInvalidOutputRoot {
			t.Fatalf("unexpected error for %q: %v", path, err)
		}
	})
}
