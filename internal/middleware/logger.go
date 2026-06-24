package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// Logger 记录每个请求的方法/路径/状态/耗时/字节数。
// statusWriter 透传 http.Flusher 以支持 SSE 流式响应。
func Logger(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			log.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.status,
				"bytes", sw.size,
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", ReqIDFrom(r.Context()),
			)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusWriter) Write(b []byte) (int, error) {
	n, err := s.ResponseWriter.Write(b)
	s.size += n
	return n, err
}

// Flush 透传 Flusher，保证 SSE 中间件链可刷新。
func (s *statusWriter) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
