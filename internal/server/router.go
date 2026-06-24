// Package server 组装路由、中间件与 HTTP 服务生命周期。
package server

import (
	"net/http"

	"github.com/hequan2017/new-api-proxy/internal/handler"
	"github.com/hequan2017/new-api-proxy/internal/middleware"
)

// buildHandler 注册路由并套用中间件链。
// 中间件顺序（外→内）：RequestID → Recovery → Logger → CORS → Auth → 业务。
func buildHandler(deps *handler.Deps) http.Handler {
	mux := http.NewServeMux()

	// 健康检查 / 服务信息
	mux.HandleFunc("GET /healthz", deps.Healthz)
	mux.HandleFunc("GET /", deps.Root)

	// OpenAI 兼容入口
	mux.HandleFunc("POST /v1/chat/completions", deps.Chat)
	mux.HandleFunc("POST /v1/embeddings", deps.Embeddings)
	mux.HandleFunc("POST /v1/images/generations", deps.CreateImage)
	mux.HandleFunc("POST /v1/videos", deps.CreateVideo)
	mux.HandleFunc("GET /v1/videos/{id}", deps.GetVideo)
	mux.HandleFunc("POST /v1/audio/speech", deps.CreateSpeech)
	mux.HandleFunc("POST /v1/audio/transcriptions", deps.CreateTranscription)

	return middleware.Chain(
		mux,
		middleware.RequestID(),
		middleware.Recovery(deps.Log),
		middleware.Logger(deps.Log),
		middleware.CORS(),
		middleware.Auth(deps.Cfg.Proxy.AuthKey),
	)
}
