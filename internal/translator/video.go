package translator

import (
	"strconv"
	"strings"

	"github.com/hequan2017/new-api-proxy/internal/model"
)

// VideoTaskRequest 将 OpenAI(Sora) 视频请求转为方舟 Seedance 任务请求。
// 转换要点：
//   - prompt → content:[{type:text,text:prompt}]
//   - input[].image_url → content image_url 项（role:first_frame 首帧）
//   - size "1280x720" → resolution(短边档) + ratio(约简比)
//   - seconds "8" → duration(int)
func VideoTaskRequest(req *model.VideoRequest, arkModel string) *model.ArkVideoTaskRequest {
	content := []model.ArkContentItem{{Type: "text", Text: req.Prompt}}
	for _, in := range req.Input {
		if in.ImageURL != "" {
			content = append(content, model.ArkContentItem{
				Type:     "image_url",
				ImageURL: &model.ArkMediaURL{URL: in.ImageURL},
				Role:     "first_frame",
			})
		}
	}

	out := &model.ArkVideoTaskRequest{
		Model:         arkModel,
		Content:       content,
		Watermark:     boolPtr(false),
		GenerateAudio: boolPtr(true),
	}
	if resolution, ratio, ok := sizeToResolutionRatio(req.Size); ok {
		out.Resolution, out.Ratio = resolution, ratio
	}
	if d, ok := parseSeconds(req.Seconds); ok {
		out.Duration = d
	}
	return out
}

// CreatedVideoObject 构造创建任务后的 OpenAI 响应对象。
func CreatedVideoObject(id string) *model.VideoObject {
	return &model.VideoObject{ID: id, Object: "video", Status: "queued", CreatedAt: now()}
}

// VideoObjectFromTask 将方舟任务查询结果转为 OpenAI Video 对象。
func VideoObjectFromTask(task *model.ArkVideoTask) *model.VideoObject {
	obj := &model.VideoObject{
		ID:        task.ID,
		Object:    "video",
		Status:    mapVideoStatus(task.Status),
		CreatedAt: task.CreatedAt,
	}
	if obj.Status == "failed" && len(task.Error) > 0 {
		obj.Error = &model.ErrorBody{Message: strings.TrimSpace(string(task.Error)), Type: "upstream_error"}
	}
	if task.Content != nil && task.Content.VideoURL != "" {
		obj.Output = &model.VideoOutput{VideoURL: task.Content.VideoURL}
	}
	return obj
}

// mapVideoStatus 火山状态 → OpenAI 状态。
func mapVideoStatus(s string) string {
	switch s {
	case "queued":
		return "queued"
	case "running":
		return "in_progress"
	case "succeeded":
		return "completed"
	case "failed", "expired":
		return "failed"
	default:
		return "queued"
	}
}

// sizeToResolutionRatio 由 "WxH" 推导火山 resolution（按短边档）与 ratio（约简比）。
func sizeToResolutionRatio(size string) (resolution, ratio string, ok bool) {
	w, h, ok := parseSize(size)
	if !ok {
		return "", "", false
	}
	short := w
	if h < short {
		short = h
	}
	return shortSideToResolution(short), simplifyRatio(w, h), true
}

func shortSideToResolution(short int) string {
	switch {
	case short >= 2160:
		return "4k"
	case short >= 1080:
		return "1080p"
	case short >= 720:
		return "720p"
	default:
		return "480p"
	}
}

func simplifyRatio(w, h int) string {
	g := gcd(w, h)
	return strconv.Itoa(w/g) + ":" + strconv.Itoa(h/g)
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func parseSize(s string) (w, h int, ok bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	parts := strings.Split(s, "x")
	if len(parts) != 2 {
		return 0, 0, false
	}
	wv, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	hv, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || wv <= 0 || hv <= 0 {
		return 0, 0, false
	}
	return wv, hv, true
}

func parseSeconds(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}
