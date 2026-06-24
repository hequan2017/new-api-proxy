package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/hequan2017/new-api-proxy/internal/model"
	"github.com/hequan2017/new-api-proxy/internal/pkg/httputil"
	"github.com/hequan2017/new-api-proxy/internal/translator"
)

const (
	asrMaxBody      = 32 << 20 // 32MB，multipart 上限
	asrPollInterval = 2 * time.Second
	asrPollTimeout  = 5 * time.Minute
)

// CreateTranscription POST /v1/audio/transcriptions —— OpenAI 转写 → 火山 ASR 异步任务（同步包装）。
//
// 说明：火山 ASR 需要可公网访问的音频 URL。请通过 multipart 表单字段 "url" 提供；
// 可选字段 "format" 指定音频格式（缺省按上传文件名推断，再缺省为 wav）。
// 字段 "file" 可选——仅用于推断格式，当前 MVP 不做本地→对象存储转存。
func (d *Deps) CreateTranscription(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, asrMaxBody)
	if err := r.ParseMultipartForm(asrMaxBody); err != nil {
		badRequest(w, "解析 multipart 表单失败: "+err.Error())
		return
	}

	audioURL := r.FormValue("url")
	if audioURL == "" {
		badRequest(w, "ASR 需要可访问的音频 URL，请通过表单字段 url 提供")
		return
	}

	format := r.FormValue("format")
	if format == "" {
		format = r.FormValue("response_format")
	}
	if format == "" && len(r.MultipartForm.File["file"]) > 0 {
		format = translator.InferAudioFormat(r.MultipartForm.File["file"][0].Filename)
	}
	if format == "" {
		format = "wav"
	}

	ctx, cancel := context.WithTimeout(r.Context(), asrPollTimeout)
	defer cancel()

	taskID, err := d.Speech.SubmitASR(ctx, audioURL, format)
	if err != nil {
		httputil.WriteError(w, err)
		return
	}

	// 轮询直到终态。
	for {
		status, text, err := d.Speech.QueryASR(ctx, taskID)
		if err != nil {
			httputil.WriteError(w, err)
			return
		}
		switch status {
		case "success":
			httputil.WriteJSON(w, http.StatusOK, model.TranscriptionResponse{Text: text})
			return
		case "failed":
			httputil.WriteError(w, model.NewAPIError(http.StatusBadGateway, "asr_failed", "语音识别失败"))
			return
		}

		select {
		case <-ctx.Done():
			httputil.WriteError(w, model.NewAPIError(http.StatusGatewayTimeout, "asr_timeout", "语音识别超时"))
			return
		case <-time.After(asrPollInterval):
		}
	}
}
