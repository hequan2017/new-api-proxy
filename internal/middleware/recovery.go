package middleware

import (
	"log/slog"
	"net/http"

	"github.com/hequan2017/new-api-proxy/internal/model"
	"github.com/hequan2017/new-api-proxy/internal/pkg/httputil"
)

// Recovery 捕获 panic，记录日志并返回 500，避免进程崩溃。
func Recovery(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recovered",
						"error", rec,
						"method", r.Method,
						"path", r.URL.Path,
						"request_id", ReqIDFrom(r.Context()),
					)
					httputil.WriteError(w, model.NewAPIError(http.StatusInternalServerError, "internal_error", "服务器内部错误"))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
