package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a new zap logger instance
func New(level string, environment string) (*zap.Logger, error) {
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}

	var config zap.Config
	if environment == "production" {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	config.Level = zap.NewAtomicLevelAt(zapLevel)

	logger, err := config.Build(
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	)
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// NewNop creates a no-op logger for testing
func NewNop() *zap.Logger {
	return zap.NewNop()
}

// WithContext returns a logger with additional context fields
func WithContext(logger *zap.Logger, fields ...zap.Field) *zap.Logger {
	return logger.With(fields...)
}

// Fatal logs a fatal message and exits
func Fatal(logger *zap.Logger, msg string, fields ...zap.Field) {
	logger.Fatal(msg, fields...)
	os.Exit(1)
}
