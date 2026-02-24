package service

import (
	"log/slog"
	"math"

	"lucky_project/config"
)

func serviceLogger() *slog.Logger {
	if config.AppLogger != nil {
		return config.AppLogger.With("layer", "service")
	}
	if config.AppConfig == nil {
		return slog.Default().With("layer", "service")
	}

	logger := config.GetLogger()
	if logger == nil {
		return slog.Default().With("layer", "service")
	}
	return logger.With("layer", "service")
}

// bytesToMB 将字节转换为MB并保留三位小数
func bytesToMB(sizeBytes int64) float64 {
	if sizeBytes <= 0 {
		return 0
	}
	value := float64(sizeBytes) / (1024 * 1024)
	return math.Round(value*1000) / 1000
}
