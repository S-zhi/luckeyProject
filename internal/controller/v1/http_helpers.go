package v1

import (
	"errors"
	"fmt"
	"lucky_project/internal/dao"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func parseIDParam(ctx *gin.Context, paramName string) (uint, error) {
	rawID := ctx.Param(paramName)
	id, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("%w: %s", dao.ErrInvalidID, rawID)
	}
	return uint(id), nil
}

func writeHTTPError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, dao.ErrInvalidID), errors.Is(err, dao.ErrNilEntity):
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, gorm.ErrRecordNotFound):
		ctx.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
	default:
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
