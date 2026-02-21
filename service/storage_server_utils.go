package service

import (
	"encoding/json"
	"strings"
)

const storageLabelLocal = "本地存储"

func normalizeStorageServerField(raw string) string {
	value := strings.TrimSpace(raw)
	servers := make([]string, 0, 2)
	if value != "" {
		var arr []string
		if err := json.Unmarshal([]byte(value), &arr); err == nil {
			servers = append(servers, arr...)
		} else {
			var single string
			if err := json.Unmarshal([]byte(value), &single); err == nil {
				servers = append(servers, single)
			} else {
				servers = append(servers, value)
			}
		}
	}

	normalized := normalizeStorageServerList(servers)
	if len(normalized) == 0 {
		normalized = []string{storageLabelLocal}
	}
	bytes, err := json.Marshal(normalized)
	if err != nil {
		return "[\"本地存储\"]"
	}
	return string(bytes)
}

func normalizeStorageServerList(servers []string) []string {
	seen := make(map[string]struct{}, len(servers))
	result := make([]string, 0, len(servers))
	for _, server := range servers {
		label, ok := toStorageLabel(server)
		if !ok {
			continue
		}
		if _, exists := seen[label]; exists {
			continue
		}
		seen[label] = struct{}{}
		result = append(result, label)
	}
	return result
}

func toStorageLabel(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}

	switch strings.ToLower(value) {
	case StorageTargetBaiduNetdisk, "baidu", "baidu-pan", "baidu_pan", "baidupan", "pan.baidu", "百度网盘":
		return "", false
	default:
		return storageLabelLocal, true
	}
}
