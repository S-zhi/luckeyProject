package v1

import (
	"lucky_project/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CoreServerController struct{}

func NewCoreServerController() *CoreServerController {
	return &CoreServerController{}
}

// ListCoreServers handles GET /v1/core-servers
// 返回 list，每项仅包含 key/ip/port 三个字段。
func (c *CoreServerController) ListCoreServers(ctx *gin.Context) {
	result, err := service.ListCoreServers(ctx.Request.Context())
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, result)
}
