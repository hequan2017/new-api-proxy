package middleware

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
)

type ctxKey struct{}

var reqIDKey ctxKey

// RequestID 为每个请求生成/透传 X-Request-Id，写入响应头并注入 context。
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-Id")
			if id == "" {
				id = uuid()
			}
			w.Header().Set("X-Request-Id", id)
			ctx := context.WithValue(r.Context(), reqIDKey, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ReqIDFrom 从 context 取出 request id。
func ReqIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(reqIDKey).(string); ok {
		return v
	}
	return ""
}

func uuid() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
