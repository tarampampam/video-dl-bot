package yt_dlp

import (
  "bytes"
  "context"
  "fmt"
  "os/exec"
  "strings"
)

const (
  errPrefix = "yt-dlp" // error prefix for all yt-dlp errors
  binPath   = "yt-dlp" // default binary path/name, can be overridden by options
)

type (
  runner interface {
    // Run executes the command with the given arguments and waits for it to finish.
    Run(_ context.Context, bin string, args ...string) error
  }

  options struct {
    runner  runner
    binPath string
  }
  Option func(*options)
)

// Apply a list of [Option]'s and return the updated state.
func (o options) Apply(opts ...Option) options {
  for _, opt := range opts {
    opt(&o)
  }

  return o
}

func Execute(ctx context.Context, uri string) (outErr error) {
  // wrap the error with the prefix
  defer func() {
    if outErr != nil {
      outErr = fmt.Errorf("%s: %w", errPrefix, outErr)
    }
  }()

  return nil
}

func Version(ctx context.Context, opts ...Option) (_ string, outErr error) {
  // wrap the error with the prefix
  defer func() {
    if outErr != nil {
      outErr = fmt.Errorf("%s: %w", errPrefix, outErr)
    }
  }()

  var o = options{binPath: binPath}.Apply(opts...)

}

// systemRunner is the default (system) runner for executing the gifski command line tool.
type systemRunner struct{}

var _ runner = (*systemRunner)(nil) // ensure systemRunner implements runner

func (r systemRunner) Run(ctx context.Context, binPath string, args ...string) error {
  var (
    cmd    = exec.CommandContext(ctx, binPath, args...)
    stderr = new(bytes.Buffer)
  )

  cmd.Stderr = stderr

  if err := cmd.Run(); err != nil {
    // in case if we have something in the stderr buffer, the better way is to return the error
    // with the stderr message, so we can see what went wrong
    if stderr.Len() > 0 {
      return fmt.Errorf(
        "%w: %s",
        err,
        strings.Join(strings.Split(stderr.String(), "\n"), "; "),
      )
    }

    return err
  }

  return nil
}
