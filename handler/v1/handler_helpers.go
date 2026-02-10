package v1

import (
	"errors"
	"lucky_project/dao"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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
