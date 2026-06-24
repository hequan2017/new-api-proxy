package handler

import (
	"context"
	"io"
	"net/http"

	"github.com/hequan2017/new-api-proxy/internal/model"
	"github.com/hequan2017/new-api-proxy/internal/pkg/httputil"
	"github.com/hequan2017/new-api-proxy/internal/provider/ark"
)

// relayToArk 处理基本透传接口（chat / embeddings）：
// 解析并映射 model → 透传至方舟 → 回写响应（支持 SSE 流式）。
func (d *Deps) relayToArk(w http.ResponseWriter, r *http.Request, path string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		httputil.WriteError(w, model.NewAPIError(http.StatusBadRequest, "read_body", err.Error()))
		return
	}
	defer r.Body.Close()

	newBody, stream, err := model.ResolveModelBody(body, d.Cfg.ResolveModel)
	if err != nil {
		httputil.WriteError(w, model.NewAPIError(http.StatusBadRequest, "bad_request", err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), d.Cfg.Ark.Timeout)
	defer cancel()

	resp, err := d.Ark.Relay(ctx, path, newBody)
	if err != nil {
		httputil.WriteError(w, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		httputil.WriteError(w, ark.AsError(resp))
		return
	}
	d.relayResponse(w, resp, stream)
}

// relayResponse 把上游响应透传给客户端：复用上游 Content-Type，流式时逐块 flush。
func (d *Deps) relayResponse(w http.ResponseWriter, resp *http.Response, stream bool) {
	h := w.Header()
	switch {
	case resp.Header.Get("Content-Type") != "":
		h.Set("Content-Type", resp.Header.Get("Content-Type"))
	case stream:
		h.Set("Content-Type", "text/event-stream")
	default:
		h.Set("Content-Type", "application/json")
	}
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	if err := streamCopy(w, resp.Body); err != nil {
		d.Log.Debug("relay stream copy ended", "error", err)
	}
}

// streamCopy 把 rc 内容写入 w，每次写入后 flush（若 w 实现 http.Flusher）。
func streamCopy(w http.ResponseWriter, rc io.Reader) error {
	flusher, _ := w.(http.Flusher)
	buf := make([]byte, 4096)
	for {
		n, err := rc.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return werr
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
