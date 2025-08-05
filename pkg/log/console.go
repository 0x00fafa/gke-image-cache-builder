package log

import (
	"fmt"
	"os"
	"time"
)

// ANSI color codes
const (
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
)

// Unicode icons
const (
	InfoIcon     = "ℹ️"
	WarningIcon  = "⚠️"
	ErrorIcon    = "❌"
	SuccessIcon  = "✅"
	ProgressIcon = "🔄"
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
	var color string
	var icon string
	var output *os.File = os.Stdout

	switch level {
	case LevelInfo:
		prefix = "[INFO]"
		color = Blue
		icon = InfoIcon
	case LevelWarn:
		prefix = "[WARN]"
		color = Yellow
		icon = WarningIcon
		output = os.Stderr
	case LevelError:
		prefix = "[ERROR]"
		color = Red
		icon = ErrorIcon
		output = os.Stderr
	case LevelSuccess:
		prefix = "[SUCCESS]"
		color = Green
		icon = SuccessIcon
	case LevelProgress:
		prefix = "[PROGRESS]"
		color = Cyan
		icon = ProgressIcon
	}

	fmt.Fprintf(output, "%s%s %s %s %s%s\n", color, timestamp, icon, prefix, message, Reset)
}
