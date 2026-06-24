// Package logx 初始化全局结构化日志（slog JSON）。
package logx

import (
	"log/slog"
	"os"
	"strings"
)

// Init 按 level 初始化 slog，输出 JSON 到 stdout，并设为默认 logger。
func Init(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
	slog.SetDefault(logger)
	return logger
}
