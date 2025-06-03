package bot

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var linkExtractRe = regexp.MustCompile(`(?i)\[.*?]\((https?://[^\s)]+)\)|\b(https?://\S+)|\b([\w.-]+\.[a-z]{2,})(/\S*)?`) //nolint:lll

// ExtractLink extracts a first valid URL from the given text.
// The URL must contain a path and may be with or without a protocol.
func ExtractLink(text string) (*url.URL, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("empty text provided")
	}

	matches := linkExtractRe.FindStringSubmatch(text)
	if matches == nil {
		return nil, errors.New("no link found")
	}

	var rawLink string

	switch {
	case matches[1] != "": // markdown link
		rawLink = matches[1]
	case matches[2] != "": // raw http(s) link
		rawLink = matches[2]
	case matches[3] != "": // Bare domain or domain + path
		rawLink = matches[3]
		if matches[4] != "" {
			rawLink += matches[4]
		}

		rawLink = "https://" + rawLink // default to https
	}

	u, err := url.Parse(rawLink)
	if err != nil || u.Scheme == "" || u.Host == "" || u.Path == "" {
		return nil, errors.New("invalid URL extracted")
	}

	return u, nil
}
