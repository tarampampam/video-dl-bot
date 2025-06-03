package logger_test

import (
	"testing"

	"gh.tarampamp.am/video-dl-bot/internal/logger"
)

func TestNew(t *testing.T) {
	t.Parallel()

	l, err := logger.New(logger.DebugLevel, logger.ConsoleFormat)
	assertNoError(t, err)
	assertNotNil(t, l)

	l, err = logger.New(logger.ErrorLevel, logger.JSONFormat)
	assertNoError(t, err)
	assertNotNil(t, l)
}
