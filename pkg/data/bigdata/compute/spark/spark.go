package spark

import (
	"context"
	"os/exec"
)

// Client wraps local spark-submit / spark-sql for development and testing.
// It is not a Spark Connect client; see package docs for planned remote APIs.
type Client struct {
	SparkHome string
	Master    string // e.g. "local[*]" or "spark://..."
}

func New(sparkHome string) *Client {
	return &Client{
		SparkHome: sparkHome,
		Master:    "local[*]",
	}
}

// SubmitJar submits a generic jar job via spark-submit.
func (c *Client) SubmitJar(ctx context.Context, jarPath, class string, args ...string) ([]byte, error) {
	cmdArgs := []string{
		"--class", class,
		"--master", c.Master,
		jarPath,
	}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, c.SparkHome+"/bin/spark-submit", cmdArgs...)
	return cmd.CombinedOutput()
}

// SubmitPython submits a python script via spark-submit.
func (c *Client) SubmitPython(ctx context.Context, scriptPath string, args ...string) ([]byte, error) {
	cmdArgs := []string{
		"--master", c.Master,
		scriptPath,
	}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, c.SparkHome+"/bin/spark-submit", cmdArgs...)
	return cmd.CombinedOutput()
}

// ExecuteSQL runs SQL via the spark-sql CLI (local process wrapper).
func (c *Client) ExecuteSQL(ctx context.Context, query string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, c.SparkHome+"/bin/spark-sql", "-e", query, "--master", c.Master)
	return cmd.CombinedOutput()
}
