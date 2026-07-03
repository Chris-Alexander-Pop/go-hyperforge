package validator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePathInside_SymlinkEscape(t *testing.T) {
	tempDir := t.TempDir()

	baseDir := filepath.Join(tempDir, "base")
	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create base dir: %v", err)
	}

	outsideDir := filepath.Join(tempDir, "outside")
	err = os.MkdirAll(outsideDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create outside dir: %v", err)
	}

	symlinkPath := filepath.Join(baseDir, "symlink")
	err = os.Symlink("../outside", symlinkPath)
	if err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Attempt to escape the baseDir using a symlink
	targetPath := "symlink/somefile.txt" // Evaluates to outsideDir/somefile.txt

	_, err = ValidatePathInside(baseDir, targetPath)
	if err == nil {
		t.Errorf("Expected error for path traversal attempt, got nil")
	} else if !strings.Contains(err.Error(), "path traversal attempt") {
		t.Errorf("Expected path traversal error, got: %v", err)
	}
}

func TestValidatePathInside_SymlinkValid(t *testing.T) {
	tempDir := t.TempDir()

	baseDir := filepath.Join(tempDir, "base")
	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create base dir: %v", err)
	}

	insideTarget := filepath.Join(baseDir, "inside_target")
	err = os.MkdirAll(insideTarget, 0755)
	if err != nil {
		t.Fatalf("Failed to create inside target dir: %v", err)
	}

	validSymlinkPath := filepath.Join(baseDir, "valid_symlink")
	err = os.Symlink("inside_target", validSymlinkPath)
	if err != nil {
		t.Fatalf("Failed to create valid symlink: %v", err)
	}

	targetPath := "valid_symlink/somefile.txt"

	path, err := ValidatePathInside(baseDir, targetPath)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !strings.Contains(path, "valid_symlink") {
		t.Errorf("Expected path to contain 'valid_symlink', got: %v", path)
	}
}

func TestValidatePathInside_BrokenSymlinkEscape(t *testing.T) {
	tempDir := t.TempDir()

	baseDir := filepath.Join(tempDir, "base")
	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create base dir: %v", err)
	}

	brokenSymlinkPath := filepath.Join(baseDir, "broken_symlink")
	// Points outside the base directory to a file that doesn't exist yet
	err = os.Symlink("../outside/nonexistent.txt", brokenSymlinkPath)
	if err != nil {
		t.Fatalf("Failed to create broken symlink: %v", err)
	}

	targetPath := "broken_symlink"

	_, err = ValidatePathInside(baseDir, targetPath)
	if err == nil {
		t.Errorf("Expected error for broken symlink traversal attempt, got nil")
	} else if !strings.Contains(err.Error(), "broken symlink detected") {
		t.Errorf("Expected broken symlink error, got: %v", err)
	}
}

func TestValidatePathInside_RegularFile(t *testing.T) {
	tempDir := t.TempDir()

	baseDir := filepath.Join(tempDir, "base")
	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create base dir: %v", err)
	}

	targetPath := "somefile.txt"

	path, err := ValidatePathInside(baseDir, targetPath)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !strings.Contains(path, "somefile.txt") {
		t.Errorf("Expected path to contain 'somefile.txt', got: %v", path)
	}
}
