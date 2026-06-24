package model

import "encoding/json"

// 本文件定义火山引擎（方舟 Ark + 语音技术 OpenSpeech）的原生结构。

// ============ 方舟：文生图（Seedream） ============

// ArkImageRequest 方舟 /api/v3/images/generations 请求。
type ArkImageRequest struct {
	Model                          string                 `json:"model"`
	Prompt                         string                 `json:"prompt"`
	Size                           string                 `json:"size,omitempty"`            // 2K/3K/4K 或 宽x高
	ResponseFormat                 string                 `json:"response_format,omitempty"` // url | b64_json
	OutputFormat                   string                 `json:"output_format,omitempty"`   // png | jpeg
	Watermark                      *bool                  `json:"watermark,omitempty"`       // 默认 true，代理默认置 false
	SequentialImageGeneration      string                 `json:"sequential_image_generation,omitempty"`
	SequentialImageGenerationOpts  *ArkSeqImageOpts       `json:"sequential_image_generation_options,omitempty"`
}

// ArkSeqImageOpts 组图（多图）选项。
type ArkSeqImageOpts struct {
	MaxImages int `json:"max_images,omitempty"`
}

// ArkImageResponse 方舟图片生成响应。
type ArkImageResponse struct {
	Model   string           `json:"model,omitempty"`
	Created int64            `json:"created,omitempty"`
	Data    []ArkImageData   `json:"data"`
	Usage   json.RawMessage  `json:"usage,omitempty"`
}

type ArkImageData struct {
	URL      string          `json:"url,omitempty"`
	B64JSON  string          `json:"b64_json,omitempty"`
	Size     string          `json:"size,omitempty"`
	Error    json.RawMessage `json:"error,omitempty"`
}

// ============ 方舟：文生视频（Seedance，异步任务） ============

// ArkVideoTaskRequest 方舟 /api/v3/contents/generations/tasks 创建任务请求。
type ArkVideoTaskRequest struct {
	Model         string           `json:"model"`
	Content       []ArkContentItem `json:"content"`
	Resolution    string           `json:"resolution,omitempty"`     // 480p|720p|1080p|4k
	Ratio         string           `json:"ratio,omitempty"`          // 16:9|9:16|1:1|4:3|3:4|21:9
	Duration      int              `json:"duration,omitempty"`       // 秒
	Watermark     *bool            `json:"watermark,omitempty"`
	GenerateAudio *bool            `json:"generate_audio,omitempty"`
	CallbackURL   string           `json:"callback_url,omitempty"`
}

// ArkContentItem 任务输入项。type 决定使用哪个字段。
type ArkContentItem struct {
	Type     string       `json:"type"` // text | image_url | video_url | audio_url
	Text     string       `json:"text,omitempty"`
	ImageURL *ArkMediaURL `json:"image_url,omitempty"`
	VideoURL *ArkMediaURL `json:"video_url,omitempty"`
	AudioURL *ArkMediaURL `json:"audio_url,omitempty"`
	Role     string       `json:"role,omitempty"` // first_frame | last_frame | reference_image ...
}

// ArkMediaURL 媒体引用（URL/Base64/asset://）。
type ArkMediaURL struct {
	URL string `json:"url"`
}

// ArkVideoTaskCreated 创建任务响应，仅返回任务 ID。
type ArkVideoTaskCreated struct {
	ID string `json:"id"`
}

// ArkVideoTask 查询任务响应。
type ArkVideoTask struct {
	ID        string           `json:"id"`
	Model     string           `json:"model,omitempty"`
	Status    string           `json:"status"` // queued|running|succeeded|failed|expired
	Content   *ArkVideoContent `json:"content,omitempty"`
	Usage     json.RawMessage  `json:"usage,omitempty"`
	Error     json.RawMessage  `json:"error,omitempty"`
	CreatedAt int64            `json:"created_at,omitempty"`
	UpdatedAt int64            `json:"updated_at,omitempty"`
}

type ArkVideoContent struct {
	VideoURL     string `json:"video_url,omitempty"`
	LastFrameURL string `json:"last_frame_url,omitempty"`
}

// ============ 火山语音技术：TTS（OpenSpeech） ============

// VolcTTSRequest 火山 /api/v1/tts 请求（非 OpenAI 兼容）。
type VolcTTSRequest struct {
	App     VolcTTSApp   `json:"app"`
	User    VolcTTSUser  `json:"user"`
	Audio   VolcTTSAudio `json:"audio"`
	Request VolcTTSReq   `json:"request"`
}

type VolcTTSApp struct {
	AppID   string `json:"appid"`
	Token   string `json:"token"`
	Cluster string `json:"cluster,omitempty"`
}

type VolcTTSUser struct {
	UID string `json:"uid"`
}

type VolcTTSAudio struct {
	VoiceType   string  `json:"voice_type"`
	Encoding    string  `json:"encoding"`
	SpeedRatio  float64 `json:"speed_ratio,omitempty"`
	VolumeRatio float64 `json:"volume_ratio,omitempty"`
	PitchRatio  float64 `json:"pitch_ratio,omitempty"`
}

type VolcTTSReq struct {
	ReqID     string `json:"reqid"`
	Text      string `json:"text"`
	Operation string `json:"operation"` // query(同步)
}

// VolcTTSResponse 火山 TTS 响应（含 Base64 音频）。
type VolcTTSResponse struct {
	ReqID   string `json:"reqid"`
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
	Data    string `json:"data"`
}
