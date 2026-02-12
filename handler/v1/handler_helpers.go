package v1

import (
	"errors"
	"log/slog"
	"lucky_project/config"
	"lucky_project/dao"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func handlerLogger() *slog.Logger {
	logger := config.EnsureLoggerInitialized()
	if logger == nil {
		return slog.Default().With("layer", "handler")
	}
	return logger.With("layer", "handler")
}

func writeHTTPError(ctx *gin.Context, err error) {
	logger := handlerLogger().With(
		"method", ctx.Request.Method,
		"path", ctx.FullPath(),
	)

	switch {
	case errors.Is(err, dao.ErrInvalidID), errors.Is(err, dao.ErrNilEntity), errors.Is(err, dao.ErrInvalidAction):
		logger.Warn("request failed", "status", http.StatusBadRequest, "error", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, dao.ErrAlreadyExists):
		logger.Warn("request failed", "status", http.StatusConflict, "error", err)
		ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, gorm.ErrRecordNotFound):
		logger.Warn("request failed", "status", http.StatusNotFound, "error", err)
		ctx.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
	default:
		logger.Error("request failed", "status", http.StatusInternalServerError, "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
