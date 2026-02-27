package v1

import (
	"fmt"
	"strconv"
	"strings"

	"lucky_project/dao"
	entity2 "lucky_project/entity"

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
		return 0, fmt.Errorf("%s 不能为空", key)
	}

	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s 必须是无符号整数", key)
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
		if normalized, ok := normalizeStorageServerInput(single); ok {
			servers = append(servers, normalized)
		}
	}
	for _, server := range payload.StorageServers {
		if normalized, ok := normalizeStorageServerInput(server); ok {
			servers = append(servers, normalized)
		}
	}
	return action, servers
}

func normalizeStorageServerDisplayList(servers []string) []string {
	normalized := make([]string, 0, len(servers))
	seen := make(map[string]struct{}, len(servers))
	for _, server := range servers {
		label, ok := normalizeStorageServerLabel(server)
		if !ok {
			continue
		}
		if _, exists := seen[label]; exists {
			continue
		}
		seen[label] = struct{}{}
		normalized = append(normalized, label)
	}
	return normalized
}

func buildStorageServerResponse(id uint, servers []string) gin.H {
	normalized := normalizeStorageServerDisplayList(servers)
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

func buildModelStorageServerResponse(model *entity2.Model, servers []string) gin.H {
	if model == nil {
		return buildStorageServerResponse(0, servers)
	}

	response := buildStorageServerResponse(model.ID, servers)
	response["storage_server"] = normalizeStorageServerDisplayList(servers)
	delete(response, "storage_servers")
	response["name"] = model.Name
	response["version"] = model.Version
	response["base_model_id"] = model.BaseModelID
	response["algorithm_id"] = model.AlgorithmID
	response["impl_type"] = model.AlgorithmID // 兼容历史字段命名
	response["task_type"] = model.TaskType
	response["description"] = model.Description
	response["framework"] = model.Framework
	response["weight_size_mb"] = model.WeightSizeMB
	response["size_mb"] = model.WeightSizeMB // 兼容历史字段命名
	response["create_time"] = model.CreateTime
	response["created_at"] = model.CreateTime // 兼容历史字段命名
	response["paper"] = model.Paper
	response["params_url"] = model.ParamsURL
	response["weight_name"] = model.WeightName
	response["file_name"] = model.WeightName // 兼容历史字段命名
	return response
}

func normalizeStorageServerLabel(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}

	switch strings.ToLower(value) {
	case "baidu_netdisk", "baidu", "baidu-pan", "baidu_pan", "baidupan", "pan.baidu", "百度网盘":
		return "百度网盘", true
	case "backend", "local", "本地存储":
		return "本地存储", true
	default:
		return value, true
	}
}

func normalizeStorageServerInput(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}

	switch strings.ToLower(value) {
	case "baidu_netdisk", "baidu", "baidu-pan", "baidu_pan", "baidupan", "pan.baidu", "百度网盘":
		return "baidu_netdisk", true
	case "backend", "local", "本地存储":
		return "backend", true
	default:
		return value, true
	}
}
