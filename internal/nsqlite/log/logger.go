package log

import (
	"io"
	"log/slog"
)

// Logger is a custom structured logger on top of slog.Logger
// that logs in JSON format.
type Logger struct {
	isInitialized bool
	slogger       *slog.Logger
}

// NewLogger creates a new Logger that writes to the given writer.
// The writer is typically os.Stdout but can be any io.Writer.
func NewLogger(writer io.Writer) Logger {
	slogger := slog.New(slog.NewJSONHandler(writer, nil))
	return Logger{
		isInitialized: true,
		slogger:       slogger,
	}
}

// IsInitialized returns true if the logger is initialized using
// NewLogger function.
func (l *Logger) IsInitialized() bool {
	return l.isInitialized
}

// Info logs structured info message.
//
// Accepts a message and a list of key-value pairs to be logged.
func (l *Logger) Info(msg string, keyVals ...KV) {
	l.slogger.Info(msg, kvToArgs(keyVals...)...)
}

// InfoNs logs structured info message with a namespace.
//
// Accepts a namespace, a message, and a list of key-value pairs to
// be logged.
//
// The namespace is used to differentiate logs from different parts
// and will be included as the first key-value pair in the log.
func (l *Logger) InfoNs(namespace, msg string, keyVals ...KV) {
	l.slogger.Info(msg, kvToArgsNs(namespace, keyVals...)...)
}

// Debug logs structured debug message.
//
// Accepts a message and a list of key-value pairs to be logged.
func (l *Logger) Debug(msg string, keyVals ...KV) {
	l.slogger.Debug(msg, kvToArgs(keyVals...)...)
}

// DebugNs logs structured debug message with a namespace.
//
// Accepts a namespace, a message, and a list of key-value pairs to
// be logged.
//
// The namespace is used to differentiate logs from different parts
// and will be included as the first key-value pair in the log.
func (l *Logger) DebugNs(namespace, msg string, keyVals ...KV) {
	l.slogger.Debug(msg, kvToArgsNs(namespace, keyVals...)...)
}

// Warn logs structured warning message.
//
// Accepts a message and a list of key-value pairs to be logged.
func (l *Logger) Warn(msg string, keyVals ...KV) {
	l.slogger.Warn(msg, kvToArgs(keyVals...)...)
}

// WarnNs logs structured warning message with a namespace.
//
// Accepts a namespace, a message, and a list of key-value pairs to
// be logged.
//
// The namespace is used to differentiate logs from different parts
// and will be included as the first key-value pair in the log.
func (l *Logger) WarnNs(namespace, msg string, keyVals ...KV) {
	l.slogger.Warn(msg, kvToArgsNs(namespace, keyVals...)...)
}

// Error logs structured error message.
//
// Accepts a message and a list of key-value pairs to be logged.
func (l *Logger) Error(msg string, keyVals ...KV) {
	l.slogger.Error(msg, kvToArgs(keyVals...)...)
}

// ErrorNs logs structured error message with a namespace.
//
// Accepts a namespace, a message, and a list of key-value pairs to
// be logged.
//
// The namespace is used to differentiate logs from different parts
// and will be included as the first key-value pair in the log.
func (l *Logger) ErrorNs(namespace, msg string, keyVals ...KV) {
	l.slogger.Error(msg, kvToArgsNs(namespace, keyVals...)...)
}
