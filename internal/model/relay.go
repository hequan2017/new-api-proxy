package model

import (
	"encoding/json"
	"fmt"
)

// ResolveModelBody 解析透传 body：用 resolve 把顶层 model 字段映射为目标名，
// 并返回是否为流式请求。
//
// 用于 chat / embeddings 等开放参数透传场景——只关心 model 与 stream，
// 其余上游兼容字段原样保留，避免逐一建模，最大化透传能力。
func ResolveModelBody(body []byte, resolve func(string) string) ([]byte, bool, error) {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, false, fmt.Errorf("decode relay body: %w", err)
	}
	if resolve != nil {
		if orig, ok := m["model"].(string); ok {
			m["model"] = resolve(orig)
		}
	}
	stream, _ := m["stream"].(bool)
	out, err := json.Marshal(m)
	if err != nil {
		return nil, false, fmt.Errorf("encode relay body: %w", err)
	}
	return out, stream, nil
}
