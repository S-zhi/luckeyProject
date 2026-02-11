package v1

import (
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"
)

func parseModelMetadataUpdates(payload map[string]interface{}) (map[string]interface{}, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("request body is empty")
	}

	updates := make(map[string]interface{}, len(payload))
	var (
		hasStorageServer  bool
		hasStorageServers bool
		storageValue      interface{}
	)

	for key, value := range payload {
		switch key {
		case "id", "create_time":
			return nil, fmt.Errorf("%s is immutable", key)
		case "name":
			name, err := parseRequiredStringField(value, "name")
			if err != nil {
				return nil, err
			}
			updates["name"] = name
		case "version":
			version, err := parsePositiveFloatField(value, "version")
			if err != nil {
				return nil, err
			}
			updates["version"] = version
		case "base_model_id":
			baseModelID, err := parseUintField(value, "base_model_id")
			if err != nil {
				return nil, err
			}
			updates["base_model_id"] = baseModelID
		case "algorithm_id":
			algorithmID, err := parseNullableStringField(value, "algorithm_id")
			if err != nil {
				return nil, err
			}
			updates["algorithm_id"] = algorithmID
		case "task_type":
			taskType, err := parseRequiredStringField(value, "task_type")
			if err != nil {
				return nil, err
			}
			updates["task_type"] = taskType
		case "description":
			description, err := parseNullableStringField(value, "description")
			if err != nil {
				return nil, err
			}
			updates["description"] = description
		case "framework":
			framework, err := parseNullableStringField(value, "framework")
			if err != nil {
				return nil, err
			}
			updates["framework"] = framework
		case "weight_size_mb":
			weightSizeMB, err := parseNonNegativeFloatField(value, "weight_size_mb")
			if err != nil {
				return nil, err
			}
			updates["weight_size_mb"] = weightSizeMB
		case "paper":
			paper, err := parseNullableStringField(value, "paper")
			if err != nil {
				return nil, err
			}
			updates["paper"] = paper
		case "params_url":
			paramsURL, err := parseNullableStringField(value, "params_url")
			if err != nil {
				return nil, err
			}
			updates["params_url"] = paramsURL
		case "weight_name":
			weightName, err := parseRequiredStringField(value, "weight_name")
			if err != nil {
				return nil, err
			}
			weightName = strings.TrimSpace(filepath.Base(weightName))
			if weightName == "" || weightName == "." || weightName == string(filepath.Separator) {
				return nil, fmt.Errorf("weight_name is invalid")
			}
			updates["weight_name"] = weightName
		case "storage_server":
			if hasStorageServers {
				return nil, fmt.Errorf("storage_server and storage_servers cannot be used together")
			}
			hasStorageServer = true
			storageValue = value
		case "storage_servers":
			if hasStorageServer {
				return nil, fmt.Errorf("storage_server and storage_servers cannot be used together")
			}
			hasStorageServers = true
			storageValue = value
		default:
			return nil, fmt.Errorf("unsupported field: %s", key)
		}
	}

	if hasStorageServer || hasStorageServers {
		normalizedStorage, err := normalizeStorageServerPatchValue(storageValue)
		if err != nil {
			return nil, err
		}
		updates["storage_server"] = normalizedStorage
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updatable fields provided")
	}

	return updates, nil
}

func parseRequiredStringField(value interface{}, field string) (string, error) {
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s must be string", field)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("%s cannot be empty", field)
	}
	return text, nil
}

func parseNullableStringField(value interface{}, field string) (*string, error) {
	if value == nil {
		return nil, nil
	}

	text, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("%s must be string or null", field)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	return &text, nil
}

func parsePositiveFloatField(value interface{}, field string) (float64, error) {
	parsed, err := parseFloatField(value, field)
	if err != nil {
		return 0, err
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than 0", field)
	}
	return parsed, nil
}

func parseNonNegativeFloatField(value interface{}, field string) (float64, error) {
	parsed, err := parseFloatField(value, field)
	if err != nil {
		return 0, err
	}
	if parsed < 0 {
		return 0, fmt.Errorf("%s must be greater than or equal to 0", field)
	}
	return parsed, nil
}

func parseFloatField(value interface{}, field string) (float64, error) {
	switch typed := value.(type) {
	case float64:
		return typed, nil
	case float32:
		return float64(typed), nil
	case int:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case uint:
		return float64(typed), nil
	case uint64:
		return float64(typed), nil
	case json.Number:
		parsed, err := typed.Float64()
		if err != nil {
			return 0, fmt.Errorf("%s must be numeric", field)
		}
		return parsed, nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, fmt.Errorf("%s must be numeric", field)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("%s must be numeric", field)
	}
}

func parseUintField(value interface{}, field string) (uint, error) {
	switch typed := value.(type) {
	case float64:
		if typed < 0 || math.Trunc(typed) != typed {
			return 0, fmt.Errorf("%s must be a non-negative integer", field)
		}
		return uint(typed), nil
	case int:
		if typed < 0 {
			return 0, fmt.Errorf("%s must be a non-negative integer", field)
		}
		return uint(typed), nil
	case int64:
		if typed < 0 {
			return 0, fmt.Errorf("%s must be a non-negative integer", field)
		}
		return uint(typed), nil
	case uint:
		return typed, nil
	case uint64:
		return uint(typed), nil
	case json.Number:
		intValue, err := typed.Int64()
		if err != nil || intValue < 0 {
			return 0, fmt.Errorf("%s must be a non-negative integer", field)
		}
		return uint(intValue), nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, fmt.Errorf("%s must be a non-negative integer", field)
		}
		intValue, err := strconv.ParseUint(trimmed, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%s must be a non-negative integer", field)
		}
		return uint(intValue), nil
	default:
		return 0, fmt.Errorf("%s must be a non-negative integer", field)
	}
}

func normalizeStorageServerPatchValue(value interface{}) (string, error) {
	servers, err := parseStorageServerPatchValue(value)
	if err != nil {
		return "", err
	}
	normalized := normalizeStorageServerArray(servers)
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("storage_server encode failed: %w", err)
	}
	return string(encoded), nil
}

func parseStorageServerPatchValue(value interface{}) ([]string, error) {
	switch typed := value.(type) {
	case nil:
		return []string{}, nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return []string{}, nil
		}

		var arr []string
		if err := json.Unmarshal([]byte(trimmed), &arr); err == nil {
			return arr, nil
		}

		var single string
		if err := json.Unmarshal([]byte(trimmed), &single); err == nil {
			return []string{single}, nil
		}
		return []string{trimmed}, nil
	case []string:
		return typed, nil
	case []interface{}:
		servers := make([]string, 0, len(typed))
		for i, item := range typed {
			server, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("storage_servers[%d] must be string", i)
			}
			servers = append(servers, server)
		}
		return servers, nil
	default:
		return nil, fmt.Errorf("storage_server must be string, string array or null")
	}
}

func normalizeStorageServerArray(servers []string) []string {
	seen := make(map[string]struct{}, len(servers))
	result := make([]string, 0, len(servers))
	for _, server := range servers {
		trimmed := strings.TrimSpace(server)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
