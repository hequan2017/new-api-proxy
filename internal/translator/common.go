// Package translator 实现 OpenAI ⇄ 火山引擎 的结构互转。
//
// 全部为无副作用纯函数（输入/输出均为 model 结构），便于单测覆盖各字段映射。
package translator

import "time"

func boolPtr(v bool) *bool { return &v }

func now() int64 { return time.Now().Unix() }
