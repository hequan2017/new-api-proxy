package handler

import (
	"encoding/json"
	"net/http"

	"github.com/hequan2017/new-api-proxy/internal/model"
	"github.com/hequan2017/new-api-proxy/internal/pkg/httputil"
)

// decodeJSON 将请求体解析到 v。
func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// badRequest 写出 400 错误。
func badRequest(w http.ResponseWriter, msg string) {
	httputil.WriteError(w, model.NewAPIError(http.StatusBadRequest, "invalid_request", msg))
}
