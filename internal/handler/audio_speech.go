package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/hequan2017/new-api-proxy/internal/model"
	"github.com/hequan2017/new-api-proxy/internal/pkg/httputil"
	"github.com/hequan2017/new-api-proxy/internal/translator"
)

// CreateSpeech POST /v1/audio/speech —— OpenAI TTS → 火山语音合成。
// 直接返回二进制音频流（兼容 OpenAI 行为）。
func (d *Deps) CreateSpeech(w http.ResponseWriter, r *http.Request) {
	var req model.TTSRequest
	if err := decodeJSON(r, &req); err != nil {
		badRequest(w, "解析 TTS 请求失败: "+err.Error())
		return
	}
	if req.Input == "" {
		badRequest(w, "input 不能为空")
		return
	}

	voiceType := d.Cfg.ResolveVoice(req.Voice)
	volcReq := translator.TTSRequest(&req, voiceType)

	ctx, cancel := context.WithTimeout(r.Context(), d.Cfg.Speech.Timeout)
	defer cancel()

	audio, encoding, err := d.Speech.Synthesize(ctx, volcReq)
	if err != nil {
		httputil.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", mimeByEncoding(encoding))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(audio)
}

// mimeByEncoding 把火山音频编码映射为 HTTP Content-Type。
func mimeByEncoding(enc string) string {
	switch strings.ToLower(enc) {
	case "mp3":
		return "audio/mpeg"
	case "wav":
		return "audio/wav"
	case "ogg":
		return "audio/ogg"
	case "aac":
		return "audio/aac"
	case "flac":
		return "audio/flac"
	case "pcm":
		return "audio/pcm"
	default:
		return "audio/mpeg"
	}
}
