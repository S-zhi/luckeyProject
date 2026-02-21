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
		if label, ok := normalizeStorageServerLabel(single); ok {
			servers = append(servers, label)
		}
	}
	for _, server := range payload.StorageServers {
		if label, ok := normalizeStorageServerLabel(server); ok {
			servers = append(servers, label)
		}
	}
	return action, servers
}

func buildStorageServerResponse(id uint, servers []string) gin.H {
	normalized := make([]string, 0, len(servers))
	for _, server := range servers {
		if label, ok := normalizeStorageServerLabel(server); ok {
			normalized = append(normalized, label)
		}
	}

	primary := ""
	if len(normalized) > 0 {
		primary = normalized[0]
	}

	return gin.H{
		"id":              id,
		"storage_server":  primary,
		"storage_servers": normalized,
	}
}

func normalizeStorageServerLabel(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}

	switch strings.ToLower(value) {
	case "baidu_netdisk", "baidu", "baidu-pan", "baidu_pan", "baidupan", "pan.baidu", "百度网盘":
		return "", false
	default:
		return "本地存储", true
	}
}
