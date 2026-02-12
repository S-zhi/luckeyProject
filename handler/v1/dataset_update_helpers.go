package v1

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

func parseDatasetMetadataUpdates(payload map[string]interface{}) (map[string]interface{}, error) {
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
		case "id", "created_at":
			return nil, fmt.Errorf("%s is immutable", key)
		case "name":
			name, err := parseRequiredStringField(value, "name")
			if err != nil {
				return nil, err
			}
			updates["name"] = name
		case "description":
			description, err := parseNullableStringField(value, "description")
			if err != nil {
				return nil, err
			}
			updates["description"] = description
		case "task_type":
			taskType, err := parseRequiredStringField(value, "task_type")
			if err != nil {
				return nil, err
			}
			updates["task_type"] = taskType
		case "dataset_format":
			datasetFormat, err := parseRequiredStringField(value, "dataset_format")
			if err != nil {
				return nil, err
			}
			updates["dataset_format"] = datasetFormat
		case "dataset_path":
			datasetPath, err := parseRequiredStringField(value, "dataset_path")
			if err != nil {
				return nil, err
			}
			updates["dataset_path"] = datasetPath
		case "file_name":
			fileName, err := parseRequiredStringField(value, "file_name")
			if err != nil {
				return nil, err
			}
			fileName = strings.TrimSpace(filepath.Base(fileName))
			if fileName == "" || fileName == "." || fileName == string(filepath.Separator) {
				return nil, fmt.Errorf("file_name is invalid")
			}
			updates["file_name"] = fileName
		case "config_path":
			configPath, err := parseNullableStringField(value, "config_path")
			if err != nil {
				return nil, err
			}
			updates["config_path"] = configPath
		case "version":
			version, err := parseRequiredStringField(value, "version")
			if err != nil {
				return nil, err
			}
			updates["version"] = version
		case "num_classes":
			numClasses, err := parseNullableUintField(value, "num_classes")
			if err != nil {
				return nil, err
			}
			updates["num_classes"] = numClasses
		case "class_names":
			classNames, err := parseJSONRawField(value, "class_names")
			if err != nil {
				return nil, err
			}
			updates["class_names"] = classNames
		case "train_count":
			trainCount, err := parseNullableUintField(value, "train_count")
			if err != nil {
				return nil, err
			}
			updates["train_count"] = trainCount
		case "val_count":
			valCount, err := parseNullableUintField(value, "val_count")
			if err != nil {
				return nil, err
			}
			updates["val_count"] = valCount
		case "test_count":
			testCount, err := parseNullableUintField(value, "test_count")
			if err != nil {
				return nil, err
			}
			updates["test_count"] = testCount
		case "size_mb":
			sizeMB, err := parseNonNegativeFloatField(value, "size_mb")
			if err != nil {
				return nil, err
			}
			updates["size_mb"] = sizeMB
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

func parseNullableUintField(value interface{}, field string) (*uint, error) {
	if value == nil {
		return nil, nil
	}
	parsed, err := parseUintField(value, field)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseJSONRawField(value interface{}, field string) (json.RawMessage, error) {
	if value == nil {
		return nil, nil
	}

	switch typed := value.(type) {
	case json.RawMessage:
		if len(typed) == 0 {
			return nil, nil
		}
		if !json.Valid(typed) {
			return nil, fmt.Errorf("%s must be valid json", field)
		}
		return typed, nil
	case []byte:
		if len(typed) == 0 {
			return nil, nil
		}
		if !json.Valid(typed) {
			return nil, fmt.Errorf("%s must be valid json", field)
		}
		return json.RawMessage(typed), nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil, nil
		}
		if !json.Valid([]byte(trimmed)) {
			return nil, fmt.Errorf("%s must be valid json", field)
		}
		return json.RawMessage(trimmed), nil
	default:
		raw, err := json.Marshal(typed)
		if err != nil {
			return nil, fmt.Errorf("%s must be valid json", field)
		}
		return json.RawMessage(raw), nil
	}
}
