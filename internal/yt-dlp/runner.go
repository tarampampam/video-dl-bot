package ytdlp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

type (
	// Define a runner interface that abstracts how commands are run.
	// This allows swapping out implementations, which is useful for testing or extending behavior.
	runner interface {
		// Run executes the command with the given arguments in the provided context.
		// Returns a RunResult containing stdout/stderr streams and the duration, or an error.
		Run(_ context.Context, exe string, args ...string) (*RunResult, error)
	}

	// RunResult holds the output and metadata of a command execution.
	RunResult struct {
		Stdout, Stderr io.Reader     // output streams from the command
		Duration       time.Duration // total time the command took to execute
	}
)

// systemRunner is the default (system) runner for executing the external command.
type systemRunner struct{}

var _ runner = (*systemRunner)(nil) // compile-time assertion to ensure systemRunner implements runner interface

// Run executes the given executable with provided arguments within the given context.
// It captures both stdout and stderr, and records the time taken for execution.
func (r systemRunner) Run(ctx context.Context, exe string, args ...string) (*RunResult, error) {
	var (
		cmd            = exec.CommandContext(ctx, exe, args...)
		stdout, stderr = new(bytes.Buffer), new(bytes.Buffer)
		startedAt      = time.Now()
	)

	fmt.Println(args)

	// attach the buffers to the command's output streams
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// run the command and handle any errors
	if err := cmd.Run(); err != nil {
		// if the stderr buffer has contents, enhance the error with that output
		if stderr.Len() > 0 {
			return nil, fmt.Errorf(
				"%w: %s", // wrap the original error with stderr output
				err,
				strings.Join(strings.Split(stderr.String(), "\n"), "; "), // flatten multiline stderr
			)
		}

		// otherwise, return the error as-is
		return nil, err
	}

	// return successful result with output and duration
	return &RunResult{
		Stdout:   stdout,
		Stderr:   stderr,
		Duration: time.Since(startedAt),
	}, nil
}
