// Package handler 实现 OpenAI 兼容入口的 HTTP 处理器。
//
// 每个 handler 负责：解析 OpenAI 请求 → 调 translator 转换 → 调 provider 上游 →
// 反向转换/透传 → 写出 OpenAI 响应。共享依赖通过 Deps 注入，便于测试替换。
package handler

import (
	"log/slog"

	"github.com/hequan2017/new-api-proxy/internal/config"
	"github.com/hequan2017/new-api-proxy/internal/provider/ark"
	"github.com/hequan2017/new-api-proxy/internal/provider/speech"
)

// Deps handler 共享依赖。
type Deps struct {
	Cfg    *config.Config
	Ark    *ark.Client
	Speech *speech.Client
	Log    *slog.Logger
}
