package errors

import (
	"fmt"
	"time"

	"github.com/mosiko1234/heimdal/sensor/internal/logger"
)

// RetryConfig defines configuration for retry behavior
type RetryConfig struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	RetryableErrors []error
}

// DefaultRetryConfig returns a sensible default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
	}
}

// RetryWithBackoff executes a function with exponential backoff retry logic
func RetryWithBackoff(operation string, config RetryConfig, fn func() error) error {
	var lastErr error
	delay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			if attempt > 1 {
				logger.Info("Operation '%s' succeeded after %d attempts", operation, attempt)
			}
			return nil
		}

		lastErr = err

		// If this is the last attempt, don't wait
		if attempt == config.MaxAttempts {
			logger.Error("Operation '%s' failed after %d attempts: %v", operation, config.MaxAttempts, err)
			break
		}

		// Log retry attempt
		logger.Warn("Operation '%s' failed (attempt %d/%d): %v. Retrying in %v...",
			operation, attempt, config.MaxAttempts, err, delay)

		// Wait before retrying
		time.Sleep(delay)

		// Calculate next delay with exponential backoff
		delay = time.Duration(float64(delay) * config.BackoffFactor)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
	}

	return fmt.Errorf("operation '%s' failed after %d attempts: %w", operation, config.MaxAttempts, lastErr)
}

// Wrap wraps an error with additional context
func Wrap(err error, context string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	contextMsg := fmt.Sprintf(context, args...)
	return fmt.Errorf("%s: %w", contextMsg, err)
}

// WrapWithLog wraps an error with context and logs it
func WrapWithLog(err error, context string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	wrapped := Wrap(err, context, args...)
	logger.Error("%v", wrapped)
	return wrapped
}

// ComponentError represents an error from a specific component
type ComponentError struct {
	Component string
	Operation string
	Err       error
}

func (e *ComponentError) Error() string {
	return fmt.Sprintf("[%s] %s: %v", e.Component, e.Operation, e.Err)
}

func (e *ComponentError) Unwrap() error {
	return e.Err
}

// NewComponentError creates a new component-specific error
func NewComponentError(component, operation string, err error) error {
	return &ComponentError{
		Component: component,
		Operation: operation,
		Err:       err,
	}
}

// RecoverableError represents an error that can be recovered from
type RecoverableError struct {
	Err       error
	Retryable bool
}

func (e *RecoverableError) Error() string {
	return e.Err.Error()
}

func (e *RecoverableError) Unwrap() error {
	return e.Err
}

// NewRecoverableError creates a new recoverable error
func NewRecoverableError(err error, retryable bool) error {
	return &RecoverableError{
		Err:       err,
		Retryable: retryable,
	}
}

// IsRecoverable checks if an error is recoverable
func IsRecoverable(err error) bool {
	if err == nil {
		return false
	}

	var recErr *RecoverableError
	if As(err, &recErr) {
		return recErr.Retryable
	}

	return false
}

// As is a wrapper around errors.As for convenience
func As(err error, target interface{}) bool {
	if err == nil {
		return false
	}

	// Simple type assertion for our custom error types
	switch t := target.(type) {
	case **ComponentError:
		if ce, ok := err.(*ComponentError); ok {
			*t = ce
			return true
		}
	case **RecoverableError:
		if re, ok := err.(*RecoverableError); ok {
			*t = re
			return true
		}
	}

	return false
}

// SafeClose safely closes a resource and logs any errors
func SafeClose(closer interface{ Close() error }, resourceName string) {
	if closer == nil {
		return
	}

	if err := closer.Close(); err != nil {
		logger.Warn("Failed to close %s: %v", resourceName, err)
	}
}

// SafeCloseWithError safely closes a resource and returns any error
func SafeCloseWithError(closer interface{ Close() error }, resourceName string) error {
	if closer == nil {
		return nil
	}

	if err := closer.Close(); err != nil {
		return Wrap(err, "failed to close %s", resourceName)
	}

	return nil
}
