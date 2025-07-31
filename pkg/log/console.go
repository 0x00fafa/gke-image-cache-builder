package log

import (
	"fmt"
	"os"
	"time"
)

// ConsoleImpl implements console-only logging (no GCS)
type ConsoleImpl struct{}

// NewConsoleImpl creates a new console logger implementation
func NewConsoleImpl() *ConsoleImpl {
	return &ConsoleImpl{}
}

// Log outputs a message to the console with appropriate formatting
func (c *ConsoleImpl) Log(level LogLevel, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	var prefix string
	var output *os.File = os.Stdout

	switch level {
	case LevelInfo:
		prefix = "[INFO]"
	case LevelWarn:
		prefix = "[WARN]"
		output = os.Stderr
	case LevelError:
		prefix = "[ERROR]"
		output = os.Stderr
	case LevelSuccess:
		prefix = "[SUCCESS]"
	case LevelProgress:
		prefix = "[PROGRESS]"
	}

	fmt.Fprintf(output, "%s %s %s\n", timestamp, prefix, message)
}
