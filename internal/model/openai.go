package model

// 本文件定义 OpenAI 兼容的请求/响应结构（本代理的对外契约）。
// chat / embeddings 走透传（见 relay.go），此处仅定义需要强类型转换的接口。

// ErrorResponse OpenAI 风格错误响应。
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code,omitempty"`
}

// ============ 文生图 ============

// ImageRequest 对齐 OpenAI /v1/images/generations（DALL·E / gpt-image-1）。
type ImageRequest struct {
	Model          string  `json:"model"`
	Prompt         string  `json:"prompt"`
	N              int     `json:"n,omitempty"`
	Size           string  `json:"size,omitempty"`
	Quality        string  `json:"quality,omitempty"`
	Style          string  `json:"style,omitempty"`
	ResponseFormat string  `json:"response_format,omitempty"` // url | b64_json
	OutputFormat   string  `json:"output_format,omitempty"`   // gpt-image-1: png/jpeg/webp
	User           string  `json:"user,omitempty"`
}

// ImageResponse OpenAI /v1/images/generations 响应。
type ImageResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
}

type ImageData struct {
	URL            string `json:"url,omitempty"`
	B64JSON        string `json:"b64_json,omitempty"`
	RevisedPrompt  string `json:"revised_prompt,omitempty"`
}

// ============ 文生视频（Sora） ============

// VideoRequest 对齐 OpenAI Videos API（sora-2 等）。创建视频任务。
type VideoRequest struct {
	Model   string       `json:"model"`
	Prompt  string       `json:"prompt"`
	Size    string       `json:"size,omitempty"`    // 如 "1280x720"
	Seconds string       `json:"seconds,omitempty"` // OpenAI 用字符串，如 "8"
	N       int          `json:"n,omitempty"`
	Input   []VideoInput `json:"input,omitempty"`   // 参考图（首帧等）
	Stream  bool         `json:"stream,omitempty"`
}

// VideoInput Sora 输入引用（首帧/参考）。
type VideoInput struct {
	Type     string `json:"type"`               // image_url | generated_file
	ImageURL string `json:"image_url,omitempty"`
	FileID   string `json:"file_id,omitempty"`
}

// VideoObject OpenAI Videos 对象（创建与查询共用）。
type VideoObject struct {
	ID        string       `json:"id"`
	Object    string       `json:"object"` // 固定 "video"
	Status    string       `json:"status"` // queued | in_progress | completed | failed
	Output    *VideoOutput `json:"output,omitempty"`
	Error     *ErrorBody   `json:"error,omitempty"`
	CreatedAt int64        `json:"created_at,omitempty"`
	Seconds   int          `json:"seconds,omitempty"`
}

type VideoOutput struct {
	URL        string `json:"url,omitempty"`
	VideoURL   string `json:"video_url,omitempty"`
	PosterURL  string `json:"poster_url,omitempty"`
}

// ============ 语音合成 TTS ============

// TTSRequest 对齐 OpenAI /v1/audio/speech。
type TTSRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format,omitempty"` // mp3 | opus | aac | flac | wav | pcm
	Speed          float64 `json:"speed,omitempty"`
}

// ============ 语音识别 ASR ============

// TranscriptionResponse 对齐 OpenAI /v1/audio/transcriptions 的简单文本响应。
type TranscriptionResponse struct {
	Text string `json:"text"`
}
