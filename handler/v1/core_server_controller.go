package v1

import (
	"errors"
	"fmt"
	"lucky_project/service"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type CoreServerController struct{}

func NewCoreServerController() *CoreServerController {
	return &CoreServerController{}
}

// ListStorageServers handles GET /v1/core-servers
// 返回 list，每项仅包含 key/ip/port 三个字段。
func (c *CoreServerController) ListStorageServers(ctx *gin.Context) {
	result, err := service.ListStorageServers(ctx.Request.Context())
	if err != nil {
		writeCoreServerHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, result)
}

type coreServerUpsertPayload struct {
	Key  string `json:"key"`
	IP   string `json:"ip"`
	Port string `json:"port"`
}

// CreateStorageServer handles POST /v1/core-servers
func (c *CoreServerController) CreateStorageServer(ctx *gin.Context) {
	var payload coreServerUpsertPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := service.CreateCoreServer(
		ctx.Request.Context(),
		payload.Key,
		payload.IP,
		payload.Port,
	)
	if err != nil {
		writeCoreServerHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, created)
}

// UpdateStorageServer handles PATCH /v1/core-servers/:key
func (c *CoreServerController) UpdateStorageServer(ctx *gin.Context) {
	key, err := parseStringPathParam(ctx, "key")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var payload coreServerUpsertPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updated, err := service.UpdateCoreServer(
		ctx.Request.Context(),
		key,
		payload.Key,
		payload.IP,
		payload.Port,
	)
	if err != nil {
		writeCoreServerHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, updated)
}

// DeleteStorageServer handles DELETE /v1/core-servers/:key
func (c *CoreServerController) DeleteStorageServer(ctx *gin.Context) {
	key, err := parseStringPathParam(ctx, "key")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := service.DeleteCoreServer(ctx.Request.Context(), key); err != nil {
		writeCoreServerHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "删除成功",
		"key":     key,
	})
}

func parseStringPathParam(ctx *gin.Context, key string) (string, error) {
	raw := strings.TrimSpace(ctx.Param(key))
	if raw == "" {
		return "", fmt.Errorf("%s 不能为空", key)
	}
	return raw, nil
}

func writeCoreServerHTTPError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCoreServerKeyRequired),
		errors.Is(err, service.ErrCoreServerIPRequired),
		errors.Is(err, service.ErrCoreServerPortRequired),
		errors.Is(err, service.ErrCoreServerPortInvalid):
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrCoreServerNotFound):
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrCoreServerAlreadyExists):
		ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrRedisNotInitialized):
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	default:
		writeHTTPError(ctx, err)
	}
}
