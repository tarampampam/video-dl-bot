package filestorage

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type (
	// fileBinOptions holds configuration for the upload operation.
	fileBinOptions struct {
		client *http.Client
		binId  string
	}

	// FileBinOption defines a functional option type for customizing fileBinOptions.
	FileBinOption func(*fileBinOptions)
)

// WithFileBinHTTPClient allows customization of the HTTP client used for requests.
func WithFileBinHTTPClient(c *http.Client) FileBinOption {
	return func(opts *fileBinOptions) {
		opts.client = c
	}
}

// Apply sets default values and applies user-provided functional options.
func (o fileBinOptions) Apply(opts ...FileBinOption) fileBinOptions {
	// set default client if not provided
	if o.client == nil {
		o.client = &http.Client{
			Timeout:   60 * time.Minute,                         //nolint:mnd // should be enough for large files
			Transport: &http.Transport{ForceAttemptHTTP2: true}, // use HTTP/2 (why not?)
		}
	}

	// set default binId if not provided
	if o.binId == "" {
		o.binId = randomString(16) //nolint:mnd // assumes randomString is defined elsewhere
	}

	// apply all user-supplied options
	for _, opt := range opts {
		opt(&o)
	}

	return o
}

// UploadToFileBin uploads a file to filebin.net and locks the bin for read-only access. The size of the file is
// determined by seeking to the end of the reader.
//
// https://github.com/espebra/filebin2
//
// Returns the public URL of the uploaded file.
func UploadToFileBin( //nolint:funlen
	ctx context.Context,
	r io.ReadSeeker,
	filename string,
	opts ...FileBinOption,
) (_ string, outErr error) {
	// wrap returned error with module-specific prefix
	defer func() {
		if outErr != nil {
			outErr = fmt.Errorf("filebin.net: %w", outErr)
		}
	}()

	var (
		o         = fileBinOptions{}.Apply(opts...)                             // initialize options with defaults
		uploadURL = fmt.Sprintf("https://filebin.net/%s/%s", o.binId, filename) // construct upload URL
	)

	// calculate the size of the file to be uploaded
	fileSize, fileSizeErr := getFileSize(r)
	if fileSizeErr != nil {
		return "", fmt.Errorf("failed to determine file size: %w", fileSizeErr)
	}

	hash, hashErr := calculateSHA256Hash(r)
	if hashErr != nil {
		return "", fmt.Errorf("failed to calculate SHA256 hash: %w", hashErr)
	}

	// create HTTP POST request to upload file
	upReq, upReqErr := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, r)
	if upReqErr != nil {
		return "", upReqErr
	}

	// set appropriate headers
	upReq.Header.Set("Content-Type", "application/octet-stream")
	upReq.Header.Set("Accept", "application/json")
	upReq.Header.Set("Content-SHA256", hash)

	upReq.ContentLength = fileSize // <-- important

	// perform upload request
	upResp, upRespErr := o.client.Do(upReq)
	if upRespErr != nil {
		return "", upRespErr
	}

	defer func() { _ = upResp.Body.Close() }()

	// check if upload was successful
	if upResp.StatusCode != http.StatusCreated {
		var body []byte

		if respBody, readErr := io.ReadAll(upResp.Body); readErr == nil {
			body = respBody
		} else {
			body = []byte("failed to read response body")
		}

		return "", fmt.Errorf("unexpected status code after upload: %d (%s)", upResp.StatusCode, string(body))
	}

	_ = upResp.Body.Close()

	// lock the bin to make it read-only
	lockReq, lockReqErr := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		fmt.Sprintf("https://filebin.net/%s", o.binId),
		http.NoBody,
	)
	if lockReqErr != nil {
		return "", lockReqErr
	}

	lockReq.Header.Set("Accept", "application/json")

	// send lock request
	lockResp, lockRespErr := o.client.Do(lockReq)
	if lockRespErr != nil {
		return "", lockRespErr
	}

	_ = lockResp.Body.Close()

	// ensure bin locking was successful
	if lockResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code after locking bin: %d", lockResp.StatusCode)
	}

	// parse final public URL
	u, uErr := url.Parse(uploadURL)
	if uErr != nil {
		return "", uErr
	}

	return u.String(), nil
}

// getFileSize calculates the size of the file to be uploaded by seeking to the end of the reader.
// It resets the reader to the beginning after checking the size.
func getFileSize(r io.Seeker) (int64, error) {
	// seek to the beginning of the reader if it supports seeking
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}

	// calculate the size of the file to be uploaded
	fileSize, seekErr := r.Seek(0, io.SeekEnd)
	if seekErr != nil {
		return 0, fmt.Errorf("failed to determine file size: %w", seekErr)
	}

	// reset the reader to the beginning after checking size
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}

	return fileSize, nil
}

func calculateSHA256Hash(r io.ReadSeeker) (string, error) {
	// reset the reader to the beginning
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	// create a new SHA256 hash
	var hash = sha256.New()

	// copy the content of the reader into the hash
	if _, err := io.Copy(hash, r); err != nil {
		return "", err
	}

	// reset the reader to the beginning after calculating the hash
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	// return the hex-encoded hash
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
