package validator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectPathTraversal_Robustness(t *testing.T) {
	payloads := []string{
		"../etc/passwd",
		"..%2fetc/passwd",       // Mixed encoded
		"%2e%2e/etc/passwd",     // Mixed encoded
		"%2e%2e%2fetc/passwd",   // Fully encoded
		"..%5cetc/passwd",       // Windows backslash mixed
		"%252e%252e/etc/passwd", // Double encoded dot, single slash
	}

	for _, p := range payloads {
		if !DetectPathTraversal(p) {
			t.Errorf("Failed to detect path traversal in payload: %s", p)
		}
	}
}

func TestValidatePathInside_Symlinks(t *testing.T) {
	// Create a temporary base directory for tests
	baseDir := t.TempDir()

	// Normal path
	validPath, err := ValidatePathInside(baseDir, "valid.txt")
	if err != nil {
		t.Errorf("Unexpected error for valid path: %v", err)
	}
	if validPath == "" {
		t.Errorf("Expected valid path, got empty string")
	}

	// Traversal attempt
	_, err = ValidatePathInside(baseDir, "../outside.txt")
	if err == nil {
		t.Errorf("Expected error for traversal attempt, got none")
	}

	// Symlink pointing outside
	linkPath := filepath.Join(baseDir, "link_out")
	err = os.Symlink("/etc", linkPath)
	if err == nil { // Some environments might not allow creating symlinks
		_, err = ValidatePathInside(baseDir, "link_out/passwd")
		if err == nil {
			t.Errorf("Expected error for symlink traversal attempt, got none")
		}
	}

	// Broken symlink
	brokenPath := filepath.Join(baseDir, "broken_link")
	err = os.Symlink("/nonexistent_path_xyz123", brokenPath)
	if err == nil {
		_, err = ValidatePathInside(baseDir, "broken_link/file.txt")
		if err == nil {
			t.Errorf("Expected error for broken symlink, got none")
		}
	}
}
