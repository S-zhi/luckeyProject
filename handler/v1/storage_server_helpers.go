package v1

import (
	"fmt"
	"strconv"
	"strings"

	"lucky_project/dao"

	"github.com/gin-gonic/gin"
)

type storageServerUpdatePayload struct {
	Action         string   `json:"action"`
	StorageServer  string   `json:"storage_server"`
	StorageServers []string `json:"storage_servers"`
}

func parseUintPathParam(ctx *gin.Context, key string) (uint, error) {
	raw := strings.TrimSpace(ctx.Param(key))
	if raw == "" {
		return 0, fmt.Errorf("%s is required", key)
	}

	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an unsigned integer", key)
	}
	return uint(value), nil
}

func normalizeStorageServerPayload(payload storageServerUpdatePayload) (string, []string) {
	action := strings.ToLower(strings.TrimSpace(payload.Action))
	if action == "" {
		action = dao.StorageActionSet
	}

	servers := make([]string, 0, len(payload.StorageServers)+1)
	if single := strings.TrimSpace(payload.StorageServer); single != "" {
		servers = append(servers, single)
	}
	for _, server := range payload.StorageServers {
		if value := strings.TrimSpace(server); value != "" {
			servers = append(servers, value)
		}
	}
	return action, servers
}

func buildStorageServerResponse(id uint, servers []string) gin.H {
	primary := ""
	if len(servers) > 0 {
		primary = servers[0]
	}

	return gin.H{
		"id":              id,
		"storage_server":  primary,
		"storage_servers": servers,
	}
}
