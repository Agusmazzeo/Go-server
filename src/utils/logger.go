package utils

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
)

type contextKey string

const loggerKey = contextKey("logger")

// NewLogger initializes a single logger that can log at multiple levels.
func NewLogger(logLevel logrus.Level, logToFile bool, filePath string) *logrus.Logger {
	logger := logrus.New()

	// Set the threshold log level (e.g., Info, Warn, Error)
	logger.SetLevel(logLevel)

	// Configure output destination
	if logToFile {
		// Open or create the log file
		file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			logger.Fatal("Could not open log file:", err)
		}
		logger.SetOutput(file)
	} else {
		logger.SetOutput(os.Stdout)
	}

	// Set formatter (JSON or Text)
	logger.SetFormatter(&logrus.JSONFormatter{}) // or use &logrus.TextFormatter{}

	return logger
}

func WithLogger(ctx context.Context, logger *logrus.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func LoggerFromContext(ctx context.Context) *logrus.Logger {
	logger, ok := ctx.Value(loggerKey).(*logrus.Logger)
	if !ok {
		// Fallback to a default logger if none is found
		defaultLogger := logrus.New()
		defaultLogger.SetLevel(logrus.InfoLevel)
		defaultLogger.SetFormatter(&logrus.TextFormatter{})
		return defaultLogger
	}
	return logger
}
