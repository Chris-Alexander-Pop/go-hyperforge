package tensorflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/ai/ml/training"
	"github.com/stretchr/testify/require"
)

func TestPathTraversal_Vulnerability(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "tensorflow-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a secret file outside the job directory (in tmpDir root)
	secretContent := "This is a secret!"
	secretFile := filepath.Join(tmpDir, "secret.txt")
	err = os.WriteFile(secretFile, []byte(secretContent), 0644)
	require.NoError(t, err)

	// Create a work directory for jobs inside tmpDir
	workDir := filepath.Join(tmpDir, "jobs")
	err = os.Mkdir(workDir, 0755)
	require.NoError(t, err)

	// Create a mock python script that simply cats the first argument
	pythonMock := filepath.Join(tmpDir, "mock_python.sh")
	mockContent := `#!/bin/sh
if [ -f "$1" ]; then
  cat "$1"
else
  echo "File not found: $1"
fi
`
	err = os.WriteFile(pythonMock, []byte(mockContent), 0755)
	require.NoError(t, err)

	// Initialize the trainer with the mock script
	trainer := New(Config{
		PythonPath: pythonMock,
		WorkDir:    workDir,
	})

	// Try to access the secret file using path traversal
	// The job runs in workDir/<jobID>/, so we need ../../secret.txt
	ctx := context.Background()
	_, err = trainer.StartJob(ctx, training.JobConfig{
		Name:       "exploit-job",
		EntryPoint: "../../secret.txt",
	})
	require.Error(t, err, "StartJob should fail for path traversal")
	require.Contains(t, err.Error(), "invalid entry point", "Error message should indicate validation failure")
}

func TestValidPath(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "tensorflow-valid-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	workDir := filepath.Join(tmpDir, "jobs")
	err = os.Mkdir(workDir, 0755)
	require.NoError(t, err)

	// Create a mock python script
	pythonMock := filepath.Join(tmpDir, "mock_python.sh")
	mockContent := `#!/bin/sh
echo "OK"
`
	err = os.WriteFile(pythonMock, []byte(mockContent), 0755)
	require.NoError(t, err)

	trainer := New(Config{
		PythonPath: pythonMock,
		WorkDir:    workDir,
	})

	ctx := context.Background()

	// Valid relative path
	job, err := trainer.StartJob(ctx, training.JobConfig{
		Name:       "valid-job",
		EntryPoint: "script.py",
	})
	require.NoError(t, err)
	require.NotNil(t, job)
}
