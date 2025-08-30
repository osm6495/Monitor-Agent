package utils

import (
	"context"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Logger provides structured logging with correlation IDs
type Logger struct {
	logger *logrus.Logger
}

// LogEntry represents a structured log entry
type LogEntry struct {
	*logrus.Entry
}

// NewLogger creates a new structured logger
func NewLogger() *Logger {
	logger := logrus.New()

	// Set output to stdout for containerized environments
	logger.SetOutput(os.Stdout)

	// Set JSON formatter for structured logging
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	return &Logger{logger: logger}
}

// WithCorrelationID creates a new log entry with correlation ID
func (l *Logger) WithCorrelationID(correlationID string) *LogEntry {
	return &LogEntry{
		Entry: l.logger.WithField("correlation_id", correlationID),
	}
}

// WithContext creates a new log entry with context information
func (l *Logger) WithContext(ctx context.Context) *LogEntry {
	entry := l.logger.WithFields(logrus.Fields{
		"timestamp": time.Now().UTC(),
	})

	// Extract correlation ID from context if available
	if correlationID, ok := ctx.Value(CorrelationIDKey).(string); ok {
		entry = entry.WithField("correlation_id", correlationID)
	}

	// Extract request ID from context if available
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		entry = entry.WithField("request_id", requestID)
	}

	return &LogEntry{Entry: entry}
}

// WithFields creates a new log entry with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *LogEntry {
	return &LogEntry{
		Entry: l.logger.WithFields(logrus.Fields(fields)),
	}
}

// GenerateCorrelationID generates a new correlation ID
func GenerateCorrelationID() string {
	return uuid.New().String()
}

// ContextKey type for context keys
type ContextKey string

const (
	CorrelationIDKey ContextKey = "correlation_id"
	RequestIDKey     ContextKey = "request_id"
)

// WithCorrelationID adds correlation ID to context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

// GetCorrelationID retrieves correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if correlationID, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return correlationID
	}
	return ""
}

// LogLevel represents log levels
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)

// SetLevel sets the log level
func (l *Logger) SetLevel(level LogLevel) {
	switch level {
	case LogLevelDebug:
		l.logger.SetLevel(logrus.DebugLevel)
	case LogLevelInfo:
		l.logger.SetLevel(logrus.InfoLevel)
	case LogLevelWarn:
		l.logger.SetLevel(logrus.WarnLevel)
	case LogLevelError:
		l.logger.SetLevel(logrus.ErrorLevel)
	case LogLevelFatal:
		l.logger.SetLevel(logrus.FatalLevel)
	default:
		l.logger.SetLevel(logrus.InfoLevel)
	}
}

// Debug logs a debug message
func (le *LogEntry) Debug(msg string, fields ...map[string]interface{}) {
	le.addFields(fields...).Debug(msg)
}

// Info logs an info message
func (le *LogEntry) Info(msg string, fields ...map[string]interface{}) {
	le.addFields(fields...).Info(msg)
}

// Warn logs a warning message
func (le *LogEntry) Warn(msg string, fields ...map[string]interface{}) {
	le.addFields(fields...).Warn(msg)
}

// Error logs an error message
func (le *LogEntry) Error(msg string, fields ...map[string]interface{}) {
	le.addFields(fields...).Error(msg)
}

// Fatal logs a fatal message and exits
func (le *LogEntry) Fatal(msg string, fields ...map[string]interface{}) {
	le.addFields(fields...).Fatal(msg)
}

// addFields adds additional fields to the log entry
func (le *LogEntry) addFields(fields ...map[string]interface{}) *logrus.Entry {
	if len(fields) == 0 {
		return le.Entry
	}

	entry := le.Entry
	for _, fieldSet := range fields {
		entry = entry.WithFields(logrus.Fields(fieldSet))
	}
	return entry
}

// LogMetrics logs application metrics
func (le *LogEntry) LogMetrics(metrics map[string]interface{}) {
	le.Info("application_metrics", map[string]interface{}{
		"metrics": metrics,
		"type":    "metrics",
	})
}

// LogAPICall logs API call information
func (le *LogEntry) LogAPICall(method, url string, statusCode int, duration time.Duration, err error) {
	fields := map[string]interface{}{
		"method":      method,
		"url":         url,
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
		"type":        "api_call",
	}

	if err != nil {
		fields["error"] = err.Error()
		le.Error("api_call_failed", fields)
	} else {
		le.Info("api_call_success", fields)
	}
}

// LogDatabaseOperation logs database operation information
func (le *LogEntry) LogDatabaseOperation(operation, table string, duration time.Duration, rowsAffected int, err error) {
	fields := map[string]interface{}{
		"operation":     operation,
		"table":         table,
		"duration_ms":   duration.Milliseconds(),
		"rows_affected": rowsAffected,
		"type":          "database_operation",
	}

	if err != nil {
		fields["error"] = err.Error()
		le.Error("database_operation_failed", fields)
	} else {
		le.Info("database_operation_success", fields)
	}
}
