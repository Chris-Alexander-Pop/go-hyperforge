package validator_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
)

func TestValidatePathInside_SymlinkBypass(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir, err := os.MkdirTemp("", "symlink-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	baseDir := filepath.Join(tmpDir, "base")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("Failed to create base dir: %v", err)
	}

	outsideDir := filepath.Join(tmpDir, "outside")
	if err := os.MkdirAll(outsideDir, 0755); err != nil {
		t.Fatalf("Failed to create outside dir: %v", err)
	}

	// Symlink in base pointing to outside
	symlinkPath := filepath.Join(baseDir, "link")
	if err := os.Symlink(outsideDir, symlinkPath); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Test 1: Direct traversal using symlink
	_, err = validator.ValidatePathInside(baseDir, "link/secret.txt")
	if err == nil {
		t.Errorf("Expected error for path traversal attempt, got nil")
	} else if !strings.Contains(err.Error(), "path traversal attempt") {
		t.Errorf("Expected path traversal error, got: %v", err)
	}

	// Test 2: Normal file inside base
	validPath, err := validator.ValidatePathInside(baseDir, "safe.txt")
	if err != nil {
		t.Errorf("Unexpected error for safe path: %v", err)
	}
	if !filepath.IsAbs(validPath) {
		t.Errorf("Expected absolute path, got: %s", validPath)
	}

	// Test 3: Broken symlink
	brokenLinkPath := filepath.Join(baseDir, "broken")
	if err := os.Symlink(filepath.Join(tmpDir, "doesnotexist"), brokenLinkPath); err != nil {
		t.Fatalf("Failed to create broken symlink: %v", err)
	}
	_, err = validator.ValidatePathInside(baseDir, "broken/secret.txt")
	if err == nil {
		t.Errorf("Expected error for broken symlink, got nil")
	} else if !strings.Contains(err.Error(), "broken symlink detected") {
		t.Errorf("Expected broken symlink error, got: %v", err)
	}
}
