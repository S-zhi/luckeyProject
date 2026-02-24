package utils

import (
	"os"
	"path/filepath"
)

// EnsureDir 确保目录存在,如果没有就创建。
func EnsureDir(path string) error {
	dir := path
	if filepath.Ext(path) != "" {
		dir = filepath.Dir(path)
	}
	return os.MkdirAll(dir, 0o755)
}
