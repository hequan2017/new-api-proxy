// Package httputil 提供统一的 JSON 响应与错误写入工具。
package httputil

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/hequan2017/new-api-proxy/internal/model"
)

// WriteJSON 写出 JSON 响应。
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError 将错误归一为 OpenAI 风格错误响应写出。
// 若 err 为 *model.APIError 则沿用其状态码与信息，否则视为 500。
func WriteError(w http.ResponseWriter, err error) {
	var e *model.APIError
	if !errors.As(err, &e) {
		e = model.NewAPIError(http.StatusInternalServerError, "internal_error", err.Error())
	}
	WriteJSON(w, e.StatusCode, model.ErrorResponse{
		Error: model.ErrorBody{
			Message: e.Message,
			Type:    e.Type,
			Code:    e.Code,
		},
	})
}
