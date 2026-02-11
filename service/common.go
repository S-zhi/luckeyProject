package service

import (
	"log/slog"

	"lucky_project/config"
)

func serviceLogger() *slog.Logger {
	if config.AppLogger != nil {
		return config.AppLogger.With("layer", "service")
	}
	if config.AppConfig == nil {
		return slog.Default().With("layer", "service")
	}

	logger := config.EnsureLoggerInitialized()
	if logger == nil {
		return slog.Default().With("layer", "service")
	}
	return logger.With("layer", "service")
}
