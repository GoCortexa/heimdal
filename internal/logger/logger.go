package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// String returns the string representation of a log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel converts a string to a LogLevel
func ParseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn", "warning":
		return WARN
	case "error":
		return ERROR
	default:
		return INFO
	}
}

// Logger provides structured logging with context
type Logger struct {
	component string
	level     LogLevel
	output    io.Writer
	mu        sync.Mutex
}

var (
	globalLogger *Logger
	globalMu     sync.RWMutex
)

// Initialize sets up the global logger with file and stdout output
func Initialize(logFile string, level string) error {
	// Create log directory if it doesn't exist
	logDir := filepath.Dir(logFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file with append mode
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Create multi-writer for both stdout and file
	multiWriter := io.MultiWriter(os.Stdout, file)

	// Create global logger
	globalMu.Lock()
	globalLogger = &Logger{
		component: "main",
		level:     ParseLogLevel(level),
		output:    multiWriter,
	}
	globalMu.Unlock()

	return nil
}

// NewComponentLogger creates a new logger for a specific component
func NewComponentLogger(component string) *Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()

	if globalLogger == nil {
		// Fallback to stdout if global logger not initialized
		return &Logger{
			component: component,
			level:     INFO,
			output:    os.Stdout,
		}
	}

	return &Logger{
		component: component,
		level:     globalLogger.level,
		output:    globalLogger.output,
	}
}

// log writes a log message with the specified level
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	// Skip if below configured level
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Get caller information
	_, file, line, ok := runtime.Caller(2)
	caller := "???"
	if ok {
		caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}

	// Format timestamp
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	// Format message
	message := fmt.Sprintf(format, args...)

	// Write log entry
	logEntry := fmt.Sprintf("%s [%s] [%s] %s: %s\n",
		timestamp, level.String(), l.component, caller, message)

	l.output.Write([]byte(logEntry))
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// ErrorWithContext logs an error with additional context
func (l *Logger) ErrorWithContext(err error, context string, args ...interface{}) {
	contextMsg := fmt.Sprintf(context, args...)
	l.log(ERROR, "%s: %v", contextMsg, err)
}

// WithField returns a new logger with an additional field
func (l *Logger) WithField(key, value string) *Logger {
	return &Logger{
		component: fmt.Sprintf("%s[%s=%s]", l.component, key, value),
		level:     l.level,
		output:    l.output,
	}
}

// Global logging functions for backward compatibility
func Debug(format string, args ...interface{}) {
	globalMu.RLock()
	logger := globalLogger
	globalMu.RUnlock()

	if logger != nil {
		logger.Debug(format, args...)
	} else {
		log.Printf("[DEBUG] "+format, args...)
	}
}

func Info(format string, args ...interface{}) {
	globalMu.RLock()
	logger := globalLogger
	globalMu.RUnlock()

	if logger != nil {
		logger.Info(format, args...)
	} else {
		log.Printf("[INFO] "+format, args...)
	}
}

func Warn(format string, args ...interface{}) {
	globalMu.RLock()
	logger := globalLogger
	globalMu.RUnlock()

	if logger != nil {
		logger.Warn(format, args...)
	} else {
		log.Printf("[WARN] "+format, args...)
	}
}

func Error(format string, args ...interface{}) {
	globalMu.RLock()
	logger := globalLogger
	globalMu.RUnlock()

	if logger != nil {
		logger.Error(format, args...)
	} else {
		log.Printf("[ERROR] "+format, args...)
	}
}

func ErrorWithContext(err error, context string, args ...interface{}) {
	globalMu.RLock()
	logger := globalLogger
	globalMu.RUnlock()

	if logger != nil {
		logger.ErrorWithContext(err, context, args...)
	} else {
		contextMsg := fmt.Sprintf(context, args...)
		log.Printf("[ERROR] %s: %v", contextMsg, err)
	}
}
