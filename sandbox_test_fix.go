package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Store struct {
	baseDir string
}

func (s *Store) validatePath(key string) (string, error) {
	evalBase, err := filepath.EvalSymlinks(s.baseDir)
	if err != nil {
		return "", fmt.Errorf("invalid path: failed to evaluate base dir symlinks: %v", err)
	}
	baseDirResolved := evalBase

	// Join performs Clean implicitly
	fullPath := filepath.Join(baseDirResolved, key)

	current := fullPath
	var resolvedPath string
	var unexistingParts []string

	for {
		evalPath, err := filepath.EvalSymlinks(current)
		if err == nil {
			resolvedPath = evalPath
			break
		}

		// If we get an error, check if the path actually exists as a symlink using Lstat
		// EvalSymlinks returns an error if the TARGET of the symlink does not exist,
		// or if any component in the path does not exist.
		if _, lstatErr := os.Lstat(current); lstatErr == nil {
			// Lstat succeeded, meaning a file/symlink exists at 'current'.
			// But EvalSymlinks failed. This means 'current' is a broken symlink.
			// To prevent attackers from creating a broken symlink that points outside,
			// and then using it as a directory component, we must fail closed.
			return "", fmt.Errorf("invalid path: broken symlink detected")
		} else if !os.IsNotExist(lstatErr) {
			// Some other error occurred during Lstat (e.g. permission denied)
			return "", fmt.Errorf("invalid path: failed to lstat path: %v", lstatErr)
		}

		// At this point, Lstat failed with IsNotExist, meaning 'current' truly does not exist.
		// We walk up to the parent directory.
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("invalid path: failed to evaluate symlinks to root")
		}
		unexistingParts = append([]string{filepath.Base(current)}, unexistingParts...)
		current = parent
	}

	if len(unexistingParts) > 0 {
		resolvedPath = filepath.Join(resolvedPath, filepath.Join(unexistingParts...))
	}

	// Ensure the path is within the base directory
	// We append a separator to ensure we don't match partial directory names
	// e.g. /tmp/data vs /tmp/database
	prefix := baseDirResolved
	if !strings.HasSuffix(prefix, string(os.PathSeparator)) {
		prefix += string(os.PathSeparator)
	}

	if !strings.HasPrefix(resolvedPath, prefix) {
		return "", fmt.Errorf("invalid path: path traversal detected")
	}

	return resolvedPath, nil
}

func main() {
	// Setup
	os.MkdirAll("testdir3/base", 0755)
	os.MkdirAll("testdir3/outside", 0755)

	// Create a broken symlink pointing outside
	os.Symlink("../outside/nonexistent.txt", "testdir3/base/broken_symlink")

	s := &Store{baseDir: "testdir3/base"}

	// Test traversal using broken symlink
	path, err := s.validatePath("broken_symlink")
	fmt.Printf("path: %s, err: %v\n", path, err)
}
