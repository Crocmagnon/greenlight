// Package jsonlog implements a simple json logger.
package jsonlog

import (
	"encoding/json"
	"io"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

// Level represents log level.
type Level int8

// Pre-defined log levels.
const (
	LevelInfo Level = iota
	LevelError
	LevelFatal
	LevelOff
)

func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	case LevelOff: // satisfies linter
		return ""
	default:
		return ""
	}
}

// Properties are additional key-value context printed with the log message.
type Properties map[string]string

// Logger provides logging facilities.
// It implements [io.Writer].
type Logger struct {
	out      io.Writer
	minLevel Level
	mu       sync.Mutex
}

// New builds a ready to use Logger instance.
func New(out io.Writer, minLevel Level) *Logger {
	return &Logger{
		out:      out,
		minLevel: minLevel,
	}
}

// PrintInfo prints the message with an info level.
func (l *Logger) PrintInfo(message string, properties Properties) {
	l.print(LevelInfo, message, properties) //nolint:errcheck // we wouldn't do anything with the error.
}

// PrintError prints the error message with an error level.
func (l *Logger) PrintError(err error, properties Properties) {
	l.print(LevelError, err.Error(), properties) //nolint:errcheck // we wouldn't do anything with the error.
}

// PrintFatal prints a fatal log then exits with status code 1.
func (l *Logger) PrintFatal(err error, properties Properties) {
	l.print(LevelFatal, err.Error(), properties) //nolint:errcheck // we wouldn't do anything with the error.
	os.Exit(1)                                   //nolint:revive
}

//nolint:forbidigo
func (l *Logger) print(level Level, message string, properties Properties) (int, error) {
	if level < l.minLevel {
		return 0, nil
	}

	aux := struct {
		Level      string     `json:"level"`
		Time       string     `json:"time"`
		Message    string     `json:"message"`
		Properties Properties `json:"properties,omitempty"`
		Trace      string     `json:"trace,omitempty"`
	}{
		Level:      level.String(),
		Time:       time.Now().UTC().Format(time.RFC3339),
		Message:    message,
		Properties: properties,
	}

	if level >= LevelError {
		aux.Trace = string(debug.Stack())
	}

	var line []byte

	line, err := json.Marshal(aux)
	if err != nil {
		line = []byte(LevelError.String() + ": unable to marshal log message: " + err.Error())
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	//nolint:wrapcheck
	return l.out.Write(append(line, '\n'))
}

func (l *Logger) Write(message []byte) (int, error) {
	return l.print(LevelError, string(message), nil)
}
