package logger_test

import (
	"errors"
	"testing"

	"gh.tarampamp.am/video-dl-bot/internal/logger"
)

func TestFormat_String(t *testing.T) {
	t.Parallel()

	for name, tt := range map[string]struct {
		giveFormat logger.Format
		wantString string
	}{
		"json":      {giveFormat: logger.JSONFormat, wantString: "json"},
		"console":   {giveFormat: logger.ConsoleFormat, wantString: "console"},
		"<unknown>": {giveFormat: logger.Format(255), wantString: "format(255)"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assertEqual(t, tt.wantString, tt.giveFormat.String())
		})
	}
}

func TestParseFormat(t *testing.T) {
	t.Parallel()

	for name, tt := range map[string]struct {
		giveBytes  []byte
		giveString string
		wantFormat logger.Format
		wantError  error
	}{
		"<empty value>":          {giveBytes: []byte(""), wantFormat: logger.ConsoleFormat},
		"<empty value> (string)": {giveString: "", wantFormat: logger.ConsoleFormat},
		"console":                {giveBytes: []byte("console"), wantFormat: logger.ConsoleFormat},
		"console (string)":       {giveString: "console", wantFormat: logger.ConsoleFormat},
		"json":                   {giveBytes: []byte("json"), wantFormat: logger.JSONFormat},
		"json (string)":          {giveString: "json", wantFormat: logger.JSONFormat},
		"foobar":                 {giveBytes: []byte("foobar"), wantError: errors.New("unrecognized logging format: \"foobar\"")}, //nolint:lll
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var (
				f   logger.Format
				err error
			)

			if tt.giveString != "" {
				f, err = logger.ParseFormat(tt.giveString)
			} else {
				f, err = logger.ParseFormat(tt.giveBytes)
			}

			if tt.wantError == nil {
				assertNoError(t, err)
				assertEqual(t, tt.wantFormat, f)
			} else {
				assertErrorMessageEqual(t, err, tt.wantError.Error())
			}
		})
	}
}
