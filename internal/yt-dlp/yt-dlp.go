package ytdlp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const errPrefix = "yt-dlp" // error prefix for all yt-dlp errors

// Locate yt-dlp binary once at package initialization. Will be empty string if not found.
var exePath, _ = exec.LookPath("yt-dlp") //nolint:gochecknoglobals

// Downloaded holds metadata and file path of the downloaded video.
type Downloaded struct {
	Filepath    string        // Local path to the downloaded video file
	ID          string        // Video ID (e.g., "daOyEt3nTnY")
	Title       string        // Short title of the video
	FullTitle   string        // Full title, often includes ID or extra info
	Description string        // Description text of the video
	WebpageURL  string        // Original video URL
	MediaType   string        // Type of media (e.g., "short", "video")
	Extractor   string        // Source site or extractor (e.g., "youtube")
	Resolution  string        // e.g., "1080x1920"
	Duration    time.Duration // Duration of the video
}

type (
	// options contains runtime configuration for yt-dlp commands.
	options struct {
		runner      runner // Interface to run system commands
		exePath     string // Path to yt-dlp binary
		cookiesFile string // Path to cookies file (optional, for sites requiring authentication)

		// To download from YouTube, yt-dlp needs to solve JavaScript challenges presented by YouTube using an
		// external JavaScript runtime. This involves running challenge solver scripts maintained at yt-dlp-ejs
		// (https://github.com/yt-dlp/ejs).
		jsRuntimes string // e.g., "node", "node:/path/to/node", "bun", "deno", "quickjs"
	}

	// Option is a function that configures options.
	Option func(*options)
)

// WithRunner injects a custom command runner (useful for testing).
func WithRunner(r runner) Option { return func(o *options) { o.runner = r } }

// WithExePath sets the path to the yt-dlp executable.
func WithExePath(path string) Option { return func(o *options) { o.exePath = path } }

// WithCookiesFile sets the path to a cookies file for yt-dlp.
func WithCookiesFile(path string) Option { return func(o *options) { o.cookiesFile = path } }

// WithJSRuntimes sets the JavaScript runtimes for yt-dlp (e.g., "node", "bun", "deno", "quickjs").
func WithJSRuntimes(runtimes string) Option { return func(o *options) { o.jsRuntimes = runtimes } }

// Apply sets default values and applies any functional options.
func (o options) Apply(opts ...Option) options {
	{ // set defaults if not already provided
		switch {
		case o.exePath == "" && exePath != "":
			o.exePath = exePath // use the found yt-dlp binary path
		case o.exePath == "":
			o.exePath = "yt-dlp" // default to "yt-dlp" if not set
		}

		if o.runner == nil {
			o.runner = new(systemRunner)
		}
	}

	for _, opt := range opts {
		opt(&o)
	}

	return o
}

// Download downloads a single video from the given URL using yt-dlp.
// It writes output to a temp directory and returns structured metadata.
// The caller is responsible for cleaning up the downloaded file.
// TODO: make file size limits configurable via options.
func Download(ctx context.Context, in string, opts ...Option) (_ *Downloaded, outErr error) { //nolint:funlen
	// defer error wrapping to include module-specific prefix
	defer func() {
		if outErr != nil {
			outErr = fmt.Errorf("%s: %w", errPrefix, outErr)
		}
	}()

	// create temporary directory for download
	tmpDir, tmpErr := os.MkdirTemp("", "yt-dlp-*")
	if tmpErr != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", tmpErr)
	}

	defer func() { _ = os.RemoveAll(tmpDir) }() // clean up the temporary directory on return

	// initialize options
	var (
		o    = options{}.Apply(opts...)
		args = []string{
			// general options
			"--ignore-config",  // don't load any more configuration files except those given to --config-locations
			"--color", "never", // disable colored output
			// video selection
			"--min-filesize", "50k", // abort download if filesize is smaller than
			"--max-filesize", "2G", // abort download if filesize is larger than
			"--no-playlist", // download only the video, if the URL refers to a video and a playlist
			// download options
			"--concurrent-fragments", "1", // number of fragments of a dash/hlsnative video that should be downloaded conc-ly
			"--retries", "5", // number of retries (default is 10), or "infinite"
			"--fragment-retries", "5", // number of retries for a fragment (default is 10), or "infinite"
			"--abort-on-unavailable-fragments", // abort download if a fragment is unavailable
			// filesystem options
			"--paths", tmpDir, // set the path to the temporary directory
			// output filename template (https://github.com/yt-dlp/yt-dlp?tab=readme-ov-file#output-template)
			"--output", "result.%(ext)s",
			"--restrict-filenames",    // restrict filenames to only ASCII characters, and avoid "&" and spaces in filenames
			"--trim-filenames", "128", // limit the filename length (excluding extension)
			"--no-overwrites",           // do not overwrite any files
			"--no-continue",             // do not resume partially downloaded fragments
			"--no-mtime",                // do not use the Last-modified header to set the file modification time
			"--write-info-json",         // write video metadata to a .info.json file
			"--no-write-comments",       // do not retrieve video comments unless the extraction is known to be quick
			"--cache-dir", os.TempDir(), // where yt-dlp can store some downloaded information permanently
			// verbosity and simulation options
			"--no-progress", // do not print progress bar
			// video format options
			// https://github.com/yt-dlp/yt-dlp?tab=readme-ov-file#format-selection
			"--format", "bv*[ext=mp4][filesize<2G]+ba[ext=m4a][filesize<2G]/bv*[ext=mp4]+ba[ext=m4a]/best[filesize<2G]/best",
			"--no-post-overwrites", // do not overwrite post-processed files
			"--no-embed-info-json", // do not embed the infojson as an attachment to the video file
		}
	)

	if o.cookiesFile != "" {
		args = append(args,
			"--cookies", // Netscape formatted file to read cookies from
			o.cookiesFile,
		)
	}

	if o.jsRuntimes != "" {
		args = append(args, "--js-runtimes", o.jsRuntimes)
	}

	// run yt-dlp with selected flags
	if _, err := o.runner.Run(ctx, o.exePath, append(args, in)...); err != nil {
		return nil, fmt.Errorf("failed to download: %w", err)
	}

	// construct paths for result and metadata files
	var (
		resultFile = filepath.Join(tmpDir, "result.mp4")
		infoFile   = filepath.Join(tmpDir, "result.info.json")
	)

	// ensure the result file exists
	if _, err := os.Stat(resultFile); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("result file does not exist: %s", resultFile)
		}

		return nil, err
	}

	// ensure the metadata file exists
	if _, err := os.Stat(infoFile); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("info file does not exist: %s", infoFile)
		}

		return nil, err
	}

	var info struct {
		ID          string  `json:"id"`
		Title       string  `json:"title"`
		FullTitle   string  `json:"fulltitle"`
		Description string  `json:"description"`
		WebpageURL  string  `json:"webpage_url"`
		MediaType   string  `json:"media_type"`
		Extractor   string  `json:"extractor"`
		Resolution  string  `json:"resolution"`
		Duration    float32 `json:"duration"`
	}

	{ // open and decode metadata JSON file
		fp, fpErr := os.Open(infoFile)
		if fpErr != nil {
			return nil, fmt.Errorf("failed to open info file: %w", fpErr)
		}

		defer func() { _ = fp.Close() }() // ensure the file is closed after reading

		if err := json.NewDecoder(fp).Decode(&info); err != nil {
			return nil, fmt.Errorf("failed to decode info file: %w", err)
		}
	}

	{ // move the video to a stable temporary path (ensures file won't disappear on return)
		newTmpFile, newTmpErr := os.CreateTemp("", fmt.Sprintf("yt-dlp-result-*%s", filepath.Ext(resultFile)))
		if newTmpErr != nil {
			return nil, newTmpErr
		}

		_ = newTmpFile.Close()

		if err := os.Rename(resultFile, newTmpFile.Name()); err != nil {
			return nil, err
		}

		resultFile = newTmpFile.Name() // update result path
	}

	// return metadata and final file path
	return &Downloaded{
		Filepath:    resultFile,
		ID:          info.ID,
		Title:       info.Title,
		FullTitle:   info.FullTitle,
		Description: info.Description,
		WebpageURL:  info.WebpageURL,
		MediaType:   info.MediaType,
		Extractor:   info.Extractor,
		Resolution:  info.Resolution,
		Duration:    time.Duration(info.Duration * float32(time.Second)),
	}, nil
}

// Version returns the version of yt-dlp installed on the system.
func Version(ctx context.Context, opts ...Option) (_ string, outErr error) {
	// defer error wrapping
	defer func() {
		if outErr != nil {
			outErr = fmt.Errorf("%s: %w", errPrefix, outErr)
		}
	}()

	var o = options{}.Apply(opts...)

	res, err := o.runner.Run(ctx, o.exePath, "--version")
	if err != nil {
		return "", fmt.Errorf("failed to get version: %w", err)
	}

	stdOut, err := io.ReadAll(res.Stdout)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(stdOut)), nil
}
