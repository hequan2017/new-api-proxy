package handler

import (
	"net/http"

	"github.com/hequan2017/new-api-proxy/internal/provider/ark"
)

// Chat POST /v1/chat/completions —— 透传方舟，支持 SSE 流式。
func (d *Deps) Chat(w http.ResponseWriter, r *http.Request) {
	d.relayToArk(w, r, ark.PathChat)
}
