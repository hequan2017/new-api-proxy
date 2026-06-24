package handler

import (
	"context"
	"net/http"

	"github.com/hequan2017/new-api-proxy/internal/model"
	"github.com/hequan2017/new-api-proxy/internal/pkg/httputil"
	"github.com/hequan2017/new-api-proxy/internal/translator"
)

// CreateVideo POST /v1/videos —— OpenAI Sora → 方舟 Seedance 异步任务。
func (d *Deps) CreateVideo(w http.ResponseWriter, r *http.Request) {
	var req model.VideoRequest
	if err := decodeJSON(r, &req); err != nil {
		badRequest(w, "解析视频请求失败: "+err.Error())
		return
	}
	if req.Prompt == "" {
		badRequest(w, "prompt 不能为空")
		return
	}

	arkModel := d.Cfg.ResolveModel(req.Model)
	arkReq := translator.VideoTaskRequest(&req, arkModel)

	ctx, cancel := context.WithTimeout(r.Context(), d.Cfg.Ark.Timeout)
	defer cancel()

	id, err := d.Ark.CreateVideoTask(ctx, arkReq)
	if err != nil {
		httputil.WriteError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, translator.CreatedVideoObject(id))
}

// GetVideo GET /v1/videos/{id} —— 查询方舟视频任务，转为 OpenAI Video 对象。
func (d *Deps) GetVideo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		badRequest(w, "缺少视频任务 id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), d.Cfg.Ark.Timeout)
	defer cancel()

	task, err := d.Ark.GetVideoTask(ctx, id)
	if err != nil {
		httputil.WriteError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, translator.VideoObjectFromTask(task))
}
