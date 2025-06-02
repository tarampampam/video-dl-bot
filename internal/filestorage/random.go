package filestorage

import (
	"math/rand"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// seededRand is a rand.Rand with a seed based on the current time (enough for random strings).
var seededRand = rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec,gochecknoglobals

// stringWithCharset returns a random string of the specified length using the provided character set.
func stringWithCharset(length int, charset string) string {
	b := make([]byte, length)

	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}

	return string(b)
}

// randomString returns a random alphanumeric string of the given length.
func randomString(length int) string { return stringWithCharset(length, charset) }
