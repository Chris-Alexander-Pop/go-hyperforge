package validator_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
)

func TestValidatePathInside_Symlink(t *testing.T) {
	tmpDir := t.TempDir()

	baseDir := filepath.Join(tmpDir, "base")
	err := os.Mkdir(baseDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	outsideDir := filepath.Join(tmpDir, "outside")
	err = os.Mkdir(outsideDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(outsideDir, "secret.txt"), []byte("secret"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.Symlink(outsideDir, filepath.Join(baseDir, "link"))
	if err != nil {
		t.Fatal(err)
	}

	// This should be rejected because "link" points to "outside", which is outside "base"
	_, err = validator.ValidatePathInside(baseDir, "link/secret.txt")
	if err == nil {
		t.Errorf("expected error for symlink traversal, got nil")
	}
}

func TestValidatePathInside_BrokenSymlink(t *testing.T) {
	tmpDir := t.TempDir()

	baseDir := filepath.Join(tmpDir, "base")
	err := os.Mkdir(baseDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.Symlink(filepath.Join(tmpDir, "does-not-exist"), filepath.Join(baseDir, "broken"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = validator.ValidatePathInside(baseDir, "broken/secret.txt")
	if err == nil {
		t.Errorf("expected error for broken symlink, got nil")
	}
}
