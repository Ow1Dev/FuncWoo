package logger

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Logger struct {
	logger zerolog.Logger // Embedded zerolog logger
	file   *os.File       // Optional: file writer for logging
}

// Config holds the configuration for initializing the logger.
type Config struct {
	Writer        io.Writer      // Optional: stdout or file
	LogToFilePath string         // Optional: file path to log to (e.g., for Fluent Bit)
	Level         zerolog.Level  // Log level (DebugLevel, InfoLevel, etc.)
	AppName       string         // Application name for context
	AppVersion    string         // Application version for context
	EnableCaller  bool           // Optional: enable caller info in logs
	Hooks         []zerolog.Hook // Optional: additional zerolog hooks
	PrettyConsole bool           // Optional: use console writer if terminal
}

// InitLog initializes the global logger with the provided configuration.
func InitLog(cfg Config) Logger {
	// Determine output writer
	var writer io.Writer
	var logFile *os.File

	switch {
	case cfg.Writer != nil:
		writer = cfg.Writer

	case cfg.LogToFilePath != "":
		f, err := os.OpenFile(cfg.LogToFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "logger: failed to open log file: %v\n", err)
			writer = os.Stdout
		} else {
			writer = f
			logFile = f
		}

	default:
		writer = os.Stdout
	}

	// Enable pretty printing if desired and terminal supports it
	if cfg.PrettyConsole {
		writer = zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			w.Out = writer
		})
	}

	// Set timestamp format
	if cfg.PrettyConsole {
		zerolog.TimeFieldFormat = time.RFC3339
	} else {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	}

	// Set log level
	zerolog.SetGlobalLevel(cfg.Level)

	// Build logger with context fields
	builder := zerolog.New(writer).With().
		Timestamp().
		Str("app", cfg.AppName).
		Str("version", cfg.AppVersion)

	if cfg.EnableCaller {
		builder = builder.Caller()
	}

	logger := builder.Logger()

	// Add hooks if provided
	for _, hook := range cfg.Hooks {
		logger = logger.Hook(hook)
	}

	// Set global logger
	log.Logger = logger

	return Logger{
		logger: logger,
		file:   logFile,
	}
}

func (l *Logger) GetLogger() *zerolog.Logger {
	return &l.logger
}

func (l *Logger) Close() {
	if l.file != nil {
		if err := l.file.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close log file")
		}
	}
}
