// Package speech 封装火山引擎「语音技术」产品线（OpenSpeech）的调用：
//   - TTS 语音合成：POST {tts_url}（同步返回 Base64 音频）
//   - ASR 语音识别：submit + query 异步任务
//
// 注意：语音技术属独立产品线，鉴权与方舟不同：
//   - Header: Authorization: Bearer;{access_token}  （分号分隔，非空格）
//   - 请求体需 app{appid,token} / user{uid} / request{reqid}
//
// ASR 接口字段以实际开通产品的官方文档为准，此处用 map 灵活构造/解析，
// 便于按需校准（见各方法注释）。
package speech

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hequan2017/new-api-proxy/internal/model"
)

const (
	ttsCodeSuccess = 3000 // 火山 TTS 成功码
	uid            = "new-api-proxy"
)

// Client 语音技术 HTTP 客户端。
type Client struct {
	appID       string
	accessToken string
	ttsURL      string
	asrSubmit   string
	asrQuery    string
	http        *http.Client
}

// New 构造语音客户端。
func New(appID, accessToken, ttsURL, asrSubmitURL, asrQueryURL string) *Client {
	return &Client{
		appID:       appID,
		accessToken: accessToken,
		ttsURL:      ttsURL,
		asrSubmit:   asrSubmitURL,
		asrQuery:    asrQueryURL,
		http:        &http.Client{},
	}
}

func (c *Client) authHeader() string {
	// 火山语音鉴权：Bearer;{token}（分号）
	return "Bearer;" + c.accessToken
}

func (c *Client) ensureAuth() error {
	if c.appID == "" || c.accessToken == "" {
		return model.NewAPIError(http.StatusInternalServerError, "speech_not_configured",
			"未配置语音技术凭证（speech.app_id / speech.access_token）")
	}
	return nil
}

// Synthesize 调用 TTS，返回解码后的音频二进制与编码格式。
// 入参 in 的 audio/request 由 translator 填充，client 负责补充 app/user/reqid。
func (c *Client) Synthesize(ctx context.Context, in *model.VolcTTSRequest) (audio []byte, encoding string, err error) {
	if err := c.ensureAuth(); err != nil {
		return nil, "", err
	}
	if c.ttsURL == "" {
		return nil, "", model.NewAPIError(http.StatusInternalServerError, "tts_url_missing", "未配置 speech.tts_url")
	}
	// 补全平台必填字段。
	in.App = model.VolcTTSApp{AppID: c.appID, Token: c.accessToken, Cluster: "volcano_tts"}
	in.User = model.VolcTTSUser{UID: uid}
	if in.Request.ReqID == "" {
		in.Request.ReqID = uuid()
	}
	if in.Request.Operation == "" {
		in.Request.Operation = "query"
	}

	var resp model.VolcTTSResponse
	if err := c.doJSON(ctx, c.ttsURL, in, &resp); err != nil {
		return nil, "", err
	}
	if resp.Code != ttsCodeSuccess {
		return nil, "", model.NewAPIError(http.StatusBadGateway, "tts_failed",
			fmt.Sprintf("火山 TTS 返回错误 code=%d message=%s", resp.Code, resp.Message))
	}
	decoded, err := base64.StdEncoding.DecodeString(resp.Data)
	if err != nil {
		return nil, "", model.NewAPIError(http.StatusBadGateway, "tts_decode_failed",
			"解码 TTS 音频失败: "+err.Error())
	}
	return decoded, in.Audio.Encoding, nil
}

// SubmitASR 提交录音文件识别任务，返回任务 ID。
// audioURL 必须为火山可访问的公网/对象存储 URL；format 如 "wav"/"mp3"/"m4a"。
func (c *Client) SubmitASR(ctx context.Context, audioURL, format string) (string, error) {
	if err := c.ensureAuth(); err != nil {
		return "", err
	}
	if c.asrSubmit == "" {
		return "", model.NewAPIError(http.StatusInternalServerError, "asr_url_missing", "未配置 speech.asr_submit_url")
	}
	body := map[string]any{
		"app":    map[string]any{"appid": c.appID, "token": c.accessToken, "cluster": "volcano_bigasr"},
		"user":   map[string]any{"uid": uid},
		"audio":  map[string]any{"url": audioURL, "format": format, "source_lang": "zh"},
		"additions": map[string]any{"use_itn": true},
		"request": map[string]any{"reqid": uuid()},
	}
	var raw map[string]any
	if err := c.doJSON(ctx, c.asrSubmit, body, &raw); err != nil {
		return "", err
	}
	// 任务 ID 字段名以实际产品为准，尝试常见键。
	if id := firstString(raw, "id", "task_id", "reqid"); id != "" {
		return id, nil
	}
	return "", model.NewAPIError(http.StatusBadGateway, "asr_no_task_id",
		fmt.Sprintf("ASR 未返回任务 ID: %s", mustJSON(raw)))
}

// QueryASR 查询 ASR 任务，返回状态与识别文本。
// status 取值：success（完成）/ running（进行中）/ failed（失败）。
func (c *Client) QueryASR(ctx context.Context, taskID string) (status string, text string, err error) {
	if err := c.ensureAuth(); err != nil {
		return "", "", err
	}
	if c.asrQuery == "" {
		return "", "", model.NewAPIError(http.StatusInternalServerError, "asr_url_missing", "未配置 speech.asr_query_url")
	}
	body := map[string]any{
		"app":  map[string]any{"appid": c.appID, "token": c.accessToken, "cluster": "volcano_bigasr"},
		"request": map[string]any{"reqid": uuid(), "task_id": taskID},
	}
	var raw map[string]any
	if err := c.doJSON(ctx, c.asrQuery, body, &raw); err != nil {
		return "", "", err
	}

	// 状态判定：优先 code==1000 或显式 status 字段。
	if code, ok := toInt(raw["code"]); ok && code == 1000 {
		status = "success"
	} else if s := firstString(raw, "status", "state"); s != "" {
		status = normalizeStatus(s)
	} else {
		status = "running"
	}

	// 文本可能在顶层 text、result.text、data.text 等位置。
	text = firstString(raw, "text", "result_text")
	if text == "" {
		if r, ok := raw["result"].(map[string]any); ok {
			text = firstString(r, "text", "transcript")
		}
	}
	if text == "" {
		if d, ok := raw["data"].(map[string]any); ok {
			text = firstString(d, "text", "transcript")
		}
	}
	return status, text, nil
}

// doJSON 发送 JSON 请求并解析 2xx 响应。
func (c *Client) doJSON(ctx context.Context, url string, in any, out any) error {
	b, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return model.NewAPIError(http.StatusBadGateway, "upstream_unreachable", err.Error())
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.NewAPIError(http.StatusBadGateway, "read_upstream", err.Error())
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return model.NewAPIError(resp.StatusCode, "speech_error",
			fmt.Sprintf("语音上游返回 %d: %s", resp.StatusCode, truncate(raw)))
	}
	if out == nil {
		return nil
	}
	// out 可能是结构体或 map；统一 JSON 解析。
	if err := json.Unmarshal(raw, out); err != nil {
		return model.NewAPIError(http.StatusBadGateway, "bad_upstream_response",
			"解析语音上游响应失败: "+err.Error())
	}
	return nil
}

// ---- 工具函数 ----

func normalizeStatus(s string) string {
	switch s {
	case "success", "succeeded", "completed", "done":
		return "success"
	case "failed", "error":
		return "failed"
	default:
		return "running"
	}
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case json.Number:
		i, err := n.Int64()
		return int(i), err == nil
	}
	return 0, false
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func truncate(b []byte) string {
	const max = 256
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "..."
}

// uuid 生成 UUID v4 字符串。
func uuid() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
