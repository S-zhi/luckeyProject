package service

import (
	"encoding/json"
	"strings"
)

func normalizeStorageServerField(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "[]"
	}

	servers := make([]string, 0, 2)
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

	normalized := normalizeStorageServerList(servers)
	bytes, err := json.Marshal(normalized)
	if err != nil {
		return "[]"
	}
	return string(bytes)
}

func normalizeStorageServerList(servers []string) []string {
	seen := make(map[string]struct{}, len(servers))
	result := make([]string, 0, len(servers))
	for _, server := range servers {
		value := strings.TrimSpace(server)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
