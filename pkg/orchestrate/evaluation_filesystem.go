// SPDX-License-Identifier: MIT OR Apache-2.0
package orchestrate

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	outputRootLockFilename      = ".nanovms-evaluation.lock"
	outputRootCoordinatorSuffix = ".nanovms-evaluation.coordinator.lock"
)

// EvaluationFilesystem is the filesystem boundary used by EvaluationAction.
// OpenFile must return a real file because the output-root lock is an OS-level,
// process-safe lock.
type EvaluationFilesystem interface {
	Lstat(string) (os.FileInfo, error)
	Mkdir(string, os.FileMode) error
	MkdirAll(string, os.FileMode) error
	OpenFile(string, int, os.FileMode) (*os.File, error)
	ReadDir(string) ([]os.DirEntry, error)
	ReadFile(string) ([]byte, error)
	Remove(string) error
	AvailableSpace(string) (uint64, error)
}

type osEvaluationFilesystem struct{}

func (osEvaluationFilesystem) Lstat(path string) (os.FileInfo, error) {
	return os.Lstat(path)
}

func (osEvaluationFilesystem) Mkdir(path string, mode os.FileMode) error {
	return os.Mkdir(path, mode)
}

func (osEvaluationFilesystem) MkdirAll(path string, mode os.FileMode) error {
	return os.MkdirAll(path, mode)
}

func (osEvaluationFilesystem) OpenFile(path string, flag int, mode os.FileMode) (*os.File, error) {
	return os.OpenFile(path, flag, mode)
}

func (osEvaluationFilesystem) ReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

func (osEvaluationFilesystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (osEvaluationFilesystem) Remove(path string) error {
	return os.Remove(path)
}

func (osEvaluationFilesystem) AvailableSpace(path string) (uint64, error) {
	return outputRootAvailableSpace(path)
}

func (action *EvaluationAction) filesystem() EvaluationFilesystem {
	if action.Filesystem != nil {
		return action.Filesystem
	}
	return osEvaluationFilesystem{}
}

type preparedOutputRoot struct {
	path     string
	created  bool
	identity os.FileInfo
}

func validateOutputRootPath(path string) error {
	return validateManagedAbsolutePath(path, "output_root", CodeInvalidOutputRoot)
}

func validateManagedAbsolutePath(path, field, code string) error {
	if path == "" || strings.TrimSpace(path) == "" || strings.ContainsRune(path, '\x00') {
		return evaluationError(code, "%s must be a non-empty absolute path", field)
	}
	if !filepath.IsAbs(path) {
		return evaluationError(code, "%s must be absolute", field)
	}
	clean := filepath.Clean(path)
	if runtime.GOOS == "windows" && strings.HasPrefix(clean, `\\`) {
		return evaluationError(code, "%s must not be a UNC path", field)
	}
	if clean != path {
		return evaluationError(code, "%s must be clean and unambiguous", field)
	}
	if filepath.Dir(clean) == clean {
		return evaluationError(code, "%s must not be a filesystem root", field)
	}
	volume := filepath.VolumeName(clean)
	remainder := strings.TrimPrefix(clean, volume)
	for _, component := range strings.FieldsFunc(remainder, func(r rune) bool {
		return r == '/' || r == '\\'
	}) {
		if component == "." || component == ".." {
			return evaluationError(code, "%s must not contain traversal components", field)
		}
	}
	return nil
}

func prepareOutputRoot(filesystem EvaluationFilesystem, requested string, parentsPrepared bool) (preparedOutputRoot, error) {
	root := preparedOutputRoot{path: filepath.Clean(requested)}
	if !parentsPrepared {
		if err := prepareOutputRootParent(filesystem, root.path); err != nil {
			return root, err
		}
	}

	info, err := filesystem.Lstat(root.path)
	switch {
	case err == nil:
		if err := validatePreparedOutputRootInfo(root.path, info); err != nil {
			return root, err
		}
		root.identity = info
		return root, nil
	case !errors.Is(err, os.ErrNotExist):
		return root, evaluationError(CodeOutputRootCreateFailed, "inspect output_root: %v", err)
	default:
		if err := filesystem.Mkdir(root.path, 0o700); err != nil {
			if !errors.Is(err, fs.ErrExist) {
				return root, evaluationError(CodeOutputRootCreateFailed, "create output_root: %v", err)
			}
			info, err = filesystem.Lstat(root.path)
			if err != nil {
				return root, evaluationError(CodeOutputRootCreateFailed, "inspect concurrently created output_root: %v", err)
			}
			if err := validatePreparedOutputRootInfo(root.path, info); err != nil {
				return root, err
			}
			root.identity = info
			return root, nil
		}
		root.created = true
		info, err = filesystem.Lstat(root.path)
		if err != nil {
			return root, evaluationError(CodeOutputRootCreateFailed, "inspect prepared output_root: %v", err)
		}
		if err := validatePreparedOutputRootInfo(root.path, info); err != nil {
			return root, err
		}
		root.identity = info
		return root, nil
	}
}

func validatePreparedOutputRootInfo(path string, info os.FileInfo) error {
	if isUnsafeOutputPathEntry(info) {
		return evaluationError(CodeInvalidOutputRoot, "output_root path component %q is a symlink or reparse point", path)
	}
	if !info.IsDir() {
		return evaluationError(CodeOutputRootCollision, "output_root collides with a non-directory")
	}
	return nil
}

func prepareOutputRootParent(filesystem EvaluationFilesystem, root string) error {
	if err := validateExistingOutputPath(filesystem, root, false); err != nil {
		return err
	}
	parent := filepath.Dir(root)
	if err := filesystem.MkdirAll(parent, 0o700); err != nil {
		return evaluationError(CodeOutputRootCreateFailed, "create output_root parents: %v", err)
	}
	// Ancestors were already walked above; after MkdirAll only re-check the
	// immediate parent instead of re-walking the entire chain.
	info, err := filesystem.Lstat(parent)
	if err != nil {
		return evaluationError(CodeOutputRootCreateFailed, "inspect output_root parent %q: %v", parent, err)
	}
	if isUnsafeOutputPathEntry(info) {
		return evaluationError(CodeInvalidOutputRoot, "output_root path component %q is a symlink or reparse point", parent)
	}
	if !info.IsDir() {
		return evaluationError(CodeOutputRootCollision, "output_root parent %q is not a directory", parent)
	}
	return nil
}

func validateExistingOutputPath(filesystem EvaluationFilesystem, path string, includePath bool) error {
	current := path
	if !includePath {
		current = filepath.Dir(current)
	}
	for {
		info, err := filesystem.Lstat(current)
		if err == nil {
			if isUnsafeOutputPathEntry(info) {
				return evaluationError(CodeInvalidOutputRoot, "output_root path component %q is a symlink or reparse point", current)
			}
			if !info.IsDir() {
				return evaluationError(CodeOutputRootCollision, "output_root parent %q is not a directory", current)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return evaluationError(CodeOutputRootCreateFailed, "inspect output_root parent %q: %v", current, err)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

func cleanupCreatedOutputRoot(filesystem EvaluationFilesystem, root preparedOutputRoot, ownedLock os.FileInfo) error {
	info, err := filesystem.Lstat(root.path)
	if err != nil {
		return fmt.Errorf("inspect created output_root: %w", err)
	}
	if root.identity == nil || isUnsafeOutputPathEntry(info) || !info.IsDir() || !os.SameFile(root.identity, info) {
		return errors.New("created output_root was replaced; preserving it")
	}
	entries, err := filesystem.ReadDir(root.path)
	if err != nil {
		return fmt.Errorf("inspect created output_root contents: %w", err)
	}
	for _, entry := range entries {
		if entry.Name() != outputRootLockFilename || entry.IsDir() {
			return fmt.Errorf("created output_root contains %q; preserving it", entry.Name())
		}
	}
	if len(entries) == 1 {
		if ownedLock == nil {
			return errors.New("created output_root contains an unowned lock; preserving it")
		}
		lockPath := filepath.Join(root.path, outputRootLockFilename)
		lockInfo, err := filesystem.Lstat(lockPath)
		if err != nil {
			return fmt.Errorf("inspect output-root lock during cleanup: %w", err)
		}
		if isUnsafeOutputPathEntry(lockInfo) || !lockInfo.Mode().IsRegular() || !os.SameFile(ownedLock, lockInfo) {
			return errors.New("output-root lock identity changed; preserving it")
		}
		if err := filesystem.Remove(lockPath); err != nil {
			return fmt.Errorf("remove output-root lock: %w", err)
		}
	}
	if err := filesystem.Remove(root.path); err != nil {
		return fmt.Errorf("remove created output_root: %w", err)
	}
	return nil
}
