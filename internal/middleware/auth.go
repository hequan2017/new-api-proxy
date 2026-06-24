package middleware

import (
	"net/http"
	"strings"

	"github.com/hequan2017/new-api-proxy/internal/model"
	"github.com/hequan2017/new-api-proxy/internal/pkg/httputil"
)

// Auth 校验入口 Bearer Token。key 为空时直接放行（内网部署于 new-api 之后）。
func Auth(key string) Middleware {
	return func(next http.Handler) http.Handler {
		if key == "" {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if bearerToken(r) != key {
				httputil.WriteError(w, model.NewAPIError(http.StatusUnauthorized, "invalid_api_key", "无效的 API Key"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}
