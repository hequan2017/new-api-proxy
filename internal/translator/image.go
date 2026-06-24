package translator

import "github.com/hequan2017/new-api-proxy/internal/model"

// ImageRequest 将 OpenAI 文生图请求转为方舟 Seedream 请求。
// 转换要点：
//   - size：火山接受 "宽x高" 像素值或 "2K/3K/4K"，这里保留像素值透传；
//   - n>1：火山无 n，改用 sequential_image_generation:auto + max_images；
//   - 默认关闭水印（watermark=false）。
func ImageRequest(req *model.ImageRequest, arkModel string) *model.ArkImageRequest {
	out := &model.ArkImageRequest{
		Model:          arkModel,
		Prompt:         req.Prompt,
		Size:           req.Size, // 透传像素值
		ResponseFormat: req.ResponseFormat,
		OutputFormat:   req.OutputFormat,
		Watermark:      boolPtr(false),
	}
	if req.N > 1 {
		out.SequentialImageGeneration = "auto"
		out.SequentialImageGenerationOpts = &model.ArkSeqImageOpts{MaxImages: req.N}
	}
	return out
}

// ImageResponse 将方舟图片响应归一为 OpenAI 结构。
func ImageResponse(resp *model.ArkImageResponse) *model.ImageResponse {
	out := &model.ImageResponse{
		Created: resp.Created,
		Data:    make([]model.ImageData, 0, len(resp.Data)),
	}
	if out.Created == 0 {
		out.Created = now()
	}
	for _, d := range resp.Data {
		out.Data = append(out.Data, model.ImageData{
			URL:     d.URL,
			B64JSON: d.B64JSON,
		})
	}
	return out
}
