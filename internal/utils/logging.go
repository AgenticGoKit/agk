package utils

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// NewLogger creates a new zerolog logger with the specified level and encoder
func NewLogger(debug bool) (*zerolog.Logger, error) {
	var writer = os.Stdout
	var logger zerolog.Logger

	// Configure time format globally
	zerolog.TimeFieldFormat = time.RFC3339

	if debug {
		// Human-friendly console output for local debugging
		cw := zerolog.ConsoleWriter{Out: writer, TimeFormat: time.RFC3339}
		l := zerolog.New(cw).With().Timestamp().Logger()
		logger = l.Level(zerolog.DebugLevel)
	} else {
		// JSON output suitable for production/log aggregation
		l := zerolog.New(writer).With().Timestamp().Logger()
		logger = l.Level(zerolog.InfoLevel)
	}

	return &logger, nil
}

// NewDevelopmentLogger creates a development logger
func NewDevelopmentLogger() (*zerolog.Logger, error) {
	zerolog.TimeFieldFormat = time.RFC3339
	cw := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	l := zerolog.New(cw).With().Timestamp().Logger()
	l = l.Level(zerolog.DebugLevel)
	return &l, nil
}

// NewProductionLogger creates a production logger
func NewProductionLogger() (*zerolog.Logger, error) {
	zerolog.TimeFieldFormat = time.RFC3339
	l := zerolog.New(os.Stdout).With().Timestamp().Logger()
	l = l.Level(zerolog.InfoLevel)
	return &l, nil
}
