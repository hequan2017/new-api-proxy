package handler

import (
	"net/http"

	"github.com/hequan2017/new-api-proxy/internal/pkg/httputil"
	"github.com/hequan2017/new-api-proxy/internal/version"
)

// Healthz GET /healthz —— 容器健康检查。
func (d *Deps) Healthz(w http.ResponseWriter, r *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// Root GET / —— 服务信息。
func (d *Deps) Root(w http.ResponseWriter, r *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"service": "new-api-proxy",
		"version": version.Version,
	})
}
