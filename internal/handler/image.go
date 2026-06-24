package handler

import (
	"context"
	"net/http"

	"github.com/hequan2017/new-api-proxy/internal/model"
	"github.com/hequan2017/new-api-proxy/internal/pkg/httputil"
	"github.com/hequan2017/new-api-proxy/internal/translator"
)

// CreateImage POST /v1/images/generations —— OpenAI(DALL·E/gpt-image) → 方舟 Seedream。
func (d *Deps) CreateImage(w http.ResponseWriter, r *http.Request) {
	var req model.ImageRequest
	if err := decodeJSON(r, &req); err != nil {
		badRequest(w, "解析图片请求失败: "+err.Error())
		return
	}
	if req.Prompt == "" {
		badRequest(w, "prompt 不能为空")
		return
	}

	arkModel := d.Cfg.ResolveModel(req.Model)
	arkReq := translator.ImageRequest(&req, arkModel)

	ctx, cancel := context.WithTimeout(r.Context(), d.Cfg.Ark.Timeout)
	defer cancel()

	arkResp, err := d.Ark.CreateImage(ctx, arkReq)
	if err != nil {
		httputil.WriteError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, translator.ImageResponse(arkResp))
}
