// Package logger provides a structured logging adapter built on top of slog.
package logger

import (
	"context"
	"log/slog"
	"os"
)

// Logger defines the port for structured logging across all modules in this project.
//
// Inject this interface to decouple consumers from the concrete logging implementation.
type Logger interface {
	// Debug logs a low-level, verbose message typically used during development
	// or troubleshooting. keysAndVals are alternating key-value pairs
	// (e.g., "key1", val1, "key2", val2).
	Debug(ctx context.Context, msg string, keysAndVals ...any)

	// Info logs an informative message about normal business events
	// (e.g., server started, request completed). keysAndVals are
	// alternating key-value pairs.
	Info(ctx context.Context, msg string, keysAndVals ...any)

	// Warn logs a message about an unexpected but non-fatal situation
	// that may need attention. keysAndVals are alternating key-value pairs.
	Warn(ctx context.Context, msg string, keysAndVals ...any)

	// Error logs a message about a failure that prevented an operation
	// from completing successfully. keysAndVals are alternating key-value pairs.
	Error(ctx context.Context, msg string, keysAndVals ...any)
}

// logger is a concrete implementation of Logger backed by slog.
type logger struct {
	inner *slog.Logger
}

// NewLogger returns a new Logger that writes JSON-formatted log entries
// to os.Stdout at debug level and above.
func NewLogger() *logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	return &logger{inner: slog.New(handler)}
}

// Debug implements Logger.Debug.
func (l *logger) Debug(ctx context.Context, msg string, keysAndVals ...any) {
	l.inner.DebugContext(ctx, msg, keysAndVals...)
}

// Info implements Logger.Info.
func (l *logger) Info(ctx context.Context, msg string, keysAndVals ...any) {
	l.inner.InfoContext(ctx, msg, keysAndVals...)
}

// Warn implements Logger.Warn.
func (l *logger) Warn(ctx context.Context, msg string, keysAndVals ...any) {
	l.inner.WarnContext(ctx, msg, keysAndVals...)
}

// Error implements Logger.Error.
func (l *logger) Error(ctx context.Context, msg string, keysAndVals ...any) {
	l.inner.ErrorContext(ctx, msg, keysAndVals...)
}
