package translator

import "github.com/hequan2017/new-api-proxy/internal/model"

// TTSRequest 将 OpenAI /v1/audio/speech 请求转为火山 TTS 请求体。
// app/user/reqid 由 speech.Client 在发送时补充。
// 映射：input→text, voice→voice_type, response_format→encoding, speed→speed_ratio。
func TTSRequest(req *model.TTSRequest, voiceType string) *model.VolcTTSRequest {
	encoding := req.ResponseFormat
	if encoding == "" {
		encoding = "mp3"
	}
	out := &model.VolcTTSRequest{
		Audio: model.VolcTTSAudio{
			VoiceType: voiceType,
			Encoding:  encoding,
		},
		Request: model.VolcTTSReq{Text: req.Input},
	}
	if req.Speed > 0 {
		out.Audio.SpeedRatio = req.Speed
	}
	return out
}
