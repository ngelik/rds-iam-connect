// Package logger provides a simple logging interface for the application.
package logger

import (
	"fmt"
	"io"
	"log"
	"os"
)

// Logger provides a simple logging interface with debug capabilities.
type Logger struct {
	*log.Logger
	debug bool
}

// New creates a new Logger instance.
func New(debug bool) *Logger {
	// If debug is enabled, write to stderr, otherwise discard output
	var output = io.Discard
	if debug {
		output = os.Stderr
	}

	return &Logger{
		Logger: log.New(output, "", log.LstdFlags),
		debug:  debug,
	}
}

// Debug logs a debug message if debug mode is enabled.
func (l *Logger) Debug(v ...interface{}) {
	if l.debug {
		if err := l.Output(2, fmt.Sprint(v...)); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing debug log: %v\n", err)
		}
	}
}

// Debugf logs a formatted debug message if debug mode is enabled.
func (l *Logger) Debugf(format string, v ...interface{}) {
	if l.debug {
		if err := l.Output(2, fmt.Sprintf(format, v...)); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing debug log: %v\n", err)
		}
	}
}

// Debugln logs a debug message with a newline if debug mode is enabled.
func (l *Logger) Debugln(v ...interface{}) {
	if l.debug {
		if err := l.Output(2, fmt.Sprintln(v...)); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing debug log: %v\n", err)
		}
	}
}
