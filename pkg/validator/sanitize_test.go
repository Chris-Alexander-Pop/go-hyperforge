package validator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePathInside(t *testing.T) {
	baseDir, err := os.MkdirTemp("", "base-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(baseDir)

	outsideDir, err := os.MkdirTemp("", "outside-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outsideDir)

	symlinkPath := filepath.Join(baseDir, "link")
	if err := os.Symlink(outsideDir, symlinkPath); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		targetPath string
		expectErr  bool
	}{
		{
			name:       "valid path inside base",
			targetPath: "valid.txt",
			expectErr:  false,
		},
		{
			name:       "valid path in subfolder",
			targetPath: "sub/valid.txt",
			expectErr:  false,
		},
		{
			name:       "path traversal attempt via string",
			targetPath: "../outside.txt",
			expectErr:  true,
		},
		{
			name:       "path traversal via symlink",
			targetPath: "link/newfile.txt",
			expectErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidatePathInside(baseDir, tc.targetPath)
			if tc.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}

func TestValidatePathInsideSymlink(t *testing.T) {
	// Create a temporary base directory
	baseDir, err := os.MkdirTemp("", "base-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(baseDir)

	// Create an outside directory
	outsideDir, err := os.MkdirTemp("", "outside-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outsideDir)

	// Create a symlink inside baseDir pointing to outsideDir
	symlinkPath := filepath.Join(baseDir, "link")
	if err := os.Symlink(outsideDir, symlinkPath); err != nil {
		t.Fatal(err)
	}

	// Attempt to validate a path that goes through the symlink to a non-existent file
	targetPath := filepath.Join("link", "newfile.txt")
	result, err := ValidatePathInside(baseDir, targetPath)
	if err == nil {
		t.Fatalf("Expected error for path traversal via symlink, but got success: %s", result)
	}
}
