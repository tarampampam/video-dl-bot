package logger_test

import (
	"errors"
	"testing"

	"gh.tarampamp.am/video-dl-bot/internal/logger"
)

func TestLevel_String(t *testing.T) {
	t.Parallel()

	for name, tt := range map[string]struct {
		giveLevel  logger.Level
		wantString string
	}{
		"debug":     {giveLevel: logger.DebugLevel, wantString: "debug"},
		"info":      {giveLevel: logger.InfoLevel, wantString: "info"},
		"warn":      {giveLevel: logger.WarnLevel, wantString: "warn"},
		"error":     {giveLevel: logger.ErrorLevel, wantString: "error"},
		"<unknown>": {giveLevel: logger.Level(127), wantString: "level(127)"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assertEqual(t, tt.wantString, tt.giveLevel.String())
		})
	}
}

func TestParseLevel(t *testing.T) {
	t.Parallel()

	for name, tt := range map[string]struct {
		giveBytes  []byte
		giveString string
		wantLevel  logger.Level
		wantError  error
	}{
		"<empty value>":          {giveBytes: []byte(""), wantLevel: logger.InfoLevel},
		"<empty value> (string)": {giveString: "", wantLevel: logger.InfoLevel},
		"trace":                  {giveBytes: []byte("debug"), wantLevel: logger.DebugLevel},
		"verbose":                {giveBytes: []byte("debug"), wantLevel: logger.DebugLevel},
		"debug":                  {giveBytes: []byte("debug"), wantLevel: logger.DebugLevel},
		"debug (string)":         {giveString: "debug", wantLevel: logger.DebugLevel},
		"info":                   {giveBytes: []byte("info"), wantLevel: logger.InfoLevel},
		"warn":                   {giveBytes: []byte("warn"), wantLevel: logger.WarnLevel},
		"error":                  {giveBytes: []byte("error"), wantLevel: logger.ErrorLevel},
		"foobar":                 {giveBytes: []byte("foobar"), wantError: errors.New("unrecognized logging level: \"foobar\"")}, //nolint:lll
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var (
				l   logger.Level
				err error
			)

			if tt.giveString != "" {
				l, err = logger.ParseLevel(tt.giveString)
			} else {
				l, err = logger.ParseLevel(tt.giveBytes)
			}

			if tt.wantError == nil {
				assertNoError(t, err)
				assertEqual(t, tt.wantLevel, l)
			} else {
				assertErrorMessageEqual(t, err, tt.wantError.Error())
			}
		})
	}
}

func TestLevels(t *testing.T) {
	t.Parallel()

	assertSlicesEqual(t, []logger.Level{
		logger.DebugLevel,
		logger.InfoLevel,
		logger.WarnLevel,
		logger.ErrorLevel,
	}, logger.Levels())
}

func TestLevelStrings(t *testing.T) {
	t.Parallel()

	assertSlicesEqual(t, []string{"debug", "info", "warn", "error"}, logger.LevelStrings())
}
