package config

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	AppLogger   *slog.Logger
	loggerInitM sync.Mutex
)

func ensureLogDir(path string) error {
	// 这里假设 path 是“文件路径”或“目录路径”
	// 如果你 AppConfig.Log.Path 是目录，就 mkdir；如果是文件，就 mkdir 它的 dir
	dir := path
	if filepath.Ext(path) != "" { // 有扩展名，像 logs/app.log
		dir = filepath.Dir(path)
	}
	return os.MkdirAll(dir, 0o755)
}

func buildLogger(logPath string) *slog.Logger {
	if strings.TrimSpace(logPath) == "" {
		logPath = "logs/app.log"
	}

	// 1) 确保目录存在
	if err := ensureLogDir(logPath); err != nil {
		fmt.Printf("failed to create log directory: %v\n", err)
		return slog.Default()
	}

	// 2) 如果传进来的是目录，拼一个默认文件名
	if filepath.Ext(logPath) == "" {
		logPath = filepath.Join(logPath, "app.log")
	}

	// 3) lumberjack 轮转
	lumberjackLogger := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    100, // MB
		MaxBackups: 3,
		MaxAge:     28, // days
		Compress:   true,
	}

	// 4) stdout + file
	mw := io.MultiWriter(os.Stdout, lumberjackLogger)

	// 5) slog handler（Text 或 JSON 二选一）
	handler := slog.NewTextHandler(mw, &slog.HandlerOptions{
		Level:     slog.LevelInfo, // 最低级别：Info
		AddSource: true,           // 想要 file:line 就开；不想要就 false
	})

	logger := slog.New(handler)

	// （可选）把标准库 log 也导到同一个 mw，避免混用时丢日志
	log.SetOutput(mw)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	logger.Info("日志系统初始化成功")
	return logger
}

func logPathFromConfig() string {
	if AppConfig == nil {
		return "logs/app.log"
	}
	return strings.TrimSpace(AppConfig.Log.Path)
}

// InitLogger 使用当前配置重新初始化全局日志器。
func InitLogger() *slog.Logger {
	loggerInitM.Lock()
	defer loggerInitM.Unlock()

	AppLogger = buildLogger(logPathFromConfig())
	return AppLogger
}

// EnsureLoggerInitialized 确保全局日志器可用；若未初始化则按当前配置初始化。
func EnsureLoggerInitialized() *slog.Logger {
	loggerInitM.Lock()
	defer loggerInitM.Unlock()

	if AppLogger != nil {
		return AppLogger
	}
	AppLogger = buildLogger(logPathFromConfig())
	return AppLogger
}
