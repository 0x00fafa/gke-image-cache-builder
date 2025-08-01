package log

import (
	"fmt"
	"log"
	"os"
)

// Logger provides structured logging interface (console only, no GCS)
type Logger struct {
	verbose bool
	quiet   bool
	gcsPath string
	logger  *log.Logger
}

// LoggerImpl defines the logging implementation interface
type LoggerImpl interface {
	Log(level LogLevel, message string)
}

// LogLevel defines log levels
type LogLevel int

const (
	LevelInfo LogLevel = iota
	LevelWarn
	LevelError
	LevelSuccess
	LevelProgress
)

// NewConsoleLogger creates a console-only logger (no GCS)
func NewConsoleLogger(verbose, quiet bool) *Logger {
	return &Logger{
		verbose: verbose,
		quiet:   quiet,
		gcsPath: "",
		logger:  log.New(os.Stdout, "", log.LstdFlags),
	}
}

// NewLogger creates a new logger instance
func NewLogger(gcsPath string) *Logger {
	return &Logger{
		gcsPath: gcsPath,
		logger:  log.New(os.Stdout, "", log.LstdFlags),
	}
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	if !l.quiet {
		l.logger.Printf("[INFO] %s", msg)
	}
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Info(fmt.Sprintf(format, args...))
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	l.logger.Printf("[WARN] %s", msg)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Warn(fmt.Sprintf(format, args...))
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	l.logger.Printf("[ERROR] %s", msg)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Error(fmt.Sprintf(format, args...))
}

// Success logs a success message
func (l *Logger) Success(msg string) {
	if !l.quiet {
		l.logger.Printf("[SUCCESS] %s", msg)
	}
}

// Successf logs a formatted success message
func (l *Logger) Successf(format string, args ...interface{}) {
	l.Success(fmt.Sprintf(format, args...))
}

// Progress logs progress information
func (l *Logger) Progress(step, total int, msg string) {
	if !l.quiet {
		progressMsg := fmt.Sprintf("(%d/%d) %s", step, total, msg)
		l.logger.Printf("[PROGRESS] %s", progressMsg)
	}
}

// Progressf logs formatted progress information
func (l *Logger) Progressf(step, total int, format string, args ...interface{}) {
	l.Progress(step, total, fmt.Sprintf(format, args...))
}

// Debug logs a debug message (only in verbose mode)
func (l *Logger) Debug(msg string) {
	if l.verbose {
		l.logger.Printf("[DEBUG] %s", msg)
	}
}

// Debugf logs a formatted debug message (only in verbose mode)
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Debug(fmt.Sprintf(format, args...))
}

// ConsoleLogger is a simple console logger implementation
type ConsoleLogger struct{}

// Log outputs the message to the console
func (c *ConsoleLogger) Log(level LogLevel, message string) {
	switch level {
	case LevelInfo:
		fmt.Println("[INFO]", message)
	case LevelWarn:
		fmt.Println("[WARN]", message)
	case LevelError:
		fmt.Println("[ERROR]", message)
	case LevelSuccess:
		fmt.Println("[SUCCESS]", message)
	case LevelProgress:
		fmt.Println("[PROGRESS]", message)
	default:
		fmt.Println("[UNKNOWN]", message)
	}
}
