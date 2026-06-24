// Package middleware 提供 HTTP 中间件与链式组合。
package middleware

import "net/http"

// Middleware 标准 net/http 中间件签名。
type Middleware func(http.Handler) http.Handler

// Chain 把多个中间件由外到内作用于 h。传入顺序即外→内执行顺序。
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
