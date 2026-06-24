// Package ark 封装火山方舟（Volcano Ark）OpenAI 兼容端点的 HTTP 调用。
//
// 涵盖：文本对话/嵌入（透传，含 SSE 流式）、文生图（Seedream）、
// 文生视频任务（Seedance，创建 + 查询）。
//
// 鉴权统一使用 API Key（Authorization: Bearer <key>）。
// HTTP 客户端不设总超时，改由调用方通过 context 控制流式生命周期。
package ark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/hequan2017/new-api-proxy/internal/model"
)

// 常用上游路径（相对 baseURL，baseURL 已含 /api/v3）。
const (
	PathChat       = "/chat/completions"
	PathEmbeddings = "/embeddings"
	PathImages     = "/images/generations"
	PathVideoTask  = "/contents/generations/tasks"
)

// Client 方舟 HTTP 客户端。
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// New 构造方舟客户端。baseURL 形如 https://ark.cn-beijing.volces.com/api/v3。
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		http:    &http.Client{}, // 总超时由 context 控制，便于流式
	}
}

func (c *Client) ensureKey() error {
	if c.apiKey == "" {
		return model.NewAPIError(http.StatusInternalServerError, "ark_api_key_missing",
			"未配置 ARK API Key（ark.api_key）")
	}
	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

// Relay 透传请求（chat / embeddings）。返回上游原始响应，由调用方决定流式/非流式处理。
// 调用方负责关闭 resp.Body；非 2xx 时调用 AsError 归一错误。
func (c *Client) Relay(ctx context.Context, path string, body []byte) (*http.Response, error) {
	if err := c.ensureKey(); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, model.NewAPIError(http.StatusBadGateway, "upstream_unreachable", err.Error())
	}
	return resp, nil
}

// CreateImage 调用方舟文生图。
func (c *Client) CreateImage(ctx context.Context, req *model.ArkImageRequest) (*model.ArkImageResponse, error) {
	if err := c.ensureKey(); err != nil {
		return nil, err
	}
	var out model.ArkImageResponse
	if err := c.doJSON(ctx, http.MethodPost, PathImages, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateVideoTask 创建文生视频任务，返回任务 ID。
func (c *Client) CreateVideoTask(ctx context.Context, req *model.ArkVideoTaskRequest) (string, error) {
	if err := c.ensureKey(); err != nil {
		return "", err
	}
	var created model.ArkVideoTaskCreated
	if err := c.doJSON(ctx, http.MethodPost, PathVideoTask, req, &created); err != nil {
		return "", err
	}
	if created.ID == "" {
		return "", model.NewAPIError(http.StatusBadGateway, "empty_task_id", "上游未返回视频任务 ID")
	}
	return created.ID, nil
}

// GetVideoTask 查询视频任务状态与结果。
func (c *Client) GetVideoTask(ctx context.Context, id string) (*model.ArkVideoTask, error) {
	if err := c.ensureKey(); err != nil {
		return nil, err
	}
	var task model.ArkVideoTask
	path := fmt.Sprintf("%s/%s", PathVideoTask, url.PathEscape(id))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// doJSON 发送 JSON 请求并把 2xx 响应解析到 out。
func (c *Client) doJSON(ctx context.Context, method, path string, in any, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return model.NewAPIError(http.StatusBadGateway, "upstream_unreachable", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return AsError(resp)
	}
	if out == nil {
		return nil
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.NewAPIError(http.StatusBadGateway, "read_upstream", err.Error())
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return model.NewAPIError(http.StatusBadGateway, "bad_upstream_response",
			fmt.Sprintf("解析上游响应失败: %v; body=%s", err, truncate(raw)))
	}
	return nil
}

// AsError 把方舟非 2xx 响应归一为 APIError。
// 方舟错误体形如 {"error":{"code":"...","message":"..."}}。
func AsError(resp *http.Response) *model.APIError {
	body, _ := io.ReadAll(resp.Body)
	code, msg := "", string(body)
	var er struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &er) == nil && er.Error.Message != "" {
		msg, code = er.Error.Message, er.Error.Code
	}
	return model.NewAPIError(resp.StatusCode, code, msg)
}

func truncate(b []byte) string {
	const max = 256
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "..."
}
