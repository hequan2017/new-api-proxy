package translator

import (
	"path/filepath"
	"strings"
)

// InferAudioFormat 由文件名扩展名推断音频格式（火山 ASR 需要）。
func InferAudioFormat(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".wav":
		return "wav"
	case ".mp3":
		return "mp3"
	case ".m4a":
		return "m4a"
	case ".aac":
		return "aac"
	case ".ogg":
		return "ogg"
	case ".flac":
		return "flac"
	default:
		return "wav"
	}
}
