// Package version 存放构建期注入的版本信息（由 Makefile -ldflags 写入）。
package version

var (
	// Version 语义化版本号，默认 dev。
	Version = "dev"
	// Commit Git 短哈希。
	Commit = "none"
	// BuildTime UTC 构建时间。
	BuildTime = "unknown"
)

// String 返回单行版本摘要。
func String() string {
	return Version + " (" + Commit + ", built " + BuildTime + ")"
}
