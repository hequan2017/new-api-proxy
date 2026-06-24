# new-api-proxy 开发计划

> 版本：v1.0 ｜ 日期：2026-06-24 ｜ 状态：开发中

## 1. 项目概述

### 1.1 目标
构建一个用 **Go（最新稳定版）** 开发的 HTTP 代理服务 `new-api-proxy`，部署在 **new-api（LLM 网关）** 与 **火山引擎（火山方舟 Ark + 语音技术 OpenSpeech）** 之间：

- **上游入口**：完全兼容 **OpenAI 官方 API 格式**（含 Sora 视频格式），使 new-api 可将其直接作为一个「OpenAI 兼容渠道」对接。
- **下游出口**：将请求/响应转换为 **火山引擎** 对应产品的格式。
- **覆盖能力域**：文本对话、向量嵌入、文生图、文生视频、语音合成(TTS)、语音识别(ASR)。
- **交付形态**：单二进制 + **Docker** 部署。

### 1.2 数据流
```
 客户端 ──► new-api ──► [new-api-proxy] ──► 火山引擎
            (OpenAI格式)   (本代理:转换)     (Ark / OpenSpeech)
                OpenAI格式              火山原生格式
```

new-api 配置一个「OpenAI 渠道」，`Base URL = http://new-api-proxy:8080/v1`，`Key = 本代理鉴权Key`。本代理完成 OpenAI⇄火山的双向转换。

---

## 2. 背景与火山方舟接口调研结论

调研来源：volcengine.com 官方文档（82379 系列）+ OpenAI 官方文档。

### 2.1 火山方舟通用信息
- **Base URL（OpenAI 兼容）**：`https://ark.cn-beijing.volces.com/api/v3`
- **鉴权**：`Authorization: Bearer $ARK_API_KEY`（API Key）
- **模型标识**：`model` 字段填 **Model ID**（如 `doubao-seed-2-1-pro-32k`）或 **Endpoint ID**，非显示名。
- **地域**：默认北京（`cn-beijing`），可配置。

### 2.2 各能力域接口与兼容性

| 能力 | OpenAI 入口 | 火山目标 | 兼容度 | 转换策略 |
|---|---|---|---|---|
| 文本对话 | `POST /v1/chat/completions` | `POST /api/v3/chat/completions` | **完全兼容** | 透传 + SSE 流式透传 |
| 向量 | `POST /v1/embeddings` | `POST /api/v3/embeddings` | 完全兼容 | 透传 |
| 文生图 | `POST /v1/images/generations` | `POST /api/v3/images/generations` | 部分兼容 | 轻转换（size/n） |
| 文生视频 | `POST /v1/videos` + `GET /v1/videos/{id}` | `POST /api/v3/contents/generations/tasks` + `GET .../{id}` | 不兼容 | **重度转换** |
| 语音合成 | `POST /v1/audio/speech` | `POST openspeech.bytedance.com/api/v1/tts` | 不兼容（独立产品线） | **重度转换** |
| 语音识别 | `POST /v1/audio/transcriptions` | 火山 ASR 异步任务 | 不兼容 | **重度转换**（同步包装异步） |

> 关键约束：TTS/ASR 不在方舟 `/api/v3` 体系，属火山「语音技术」产品线，需独立 **appid + access_token**，鉴权格式为 `Authorization: Bearer;{token}`（注意是分号）。

---

## 3. 功能范围

### 3.1 必做（MVP）
1. 文本对话（含流式 SSE）
2. 文本嵌入
3. 文生图（DALL·E / gpt-image-1 → Seedream）
4. 文生视频（Sora → Seedance，含创建与轮询）
5. 语音合成（OpenAI speech → 火山 TTS）
6. 语音识别（OpenAI transcription → 火山 ASR，同步包装）
7. 模型名映射（OpenAI 模型名 → 火山 Model ID）
8. 结构化日志、错误归一化、健康检查
9. Docker 单容器部署

### 3.2 可选/后续
- TTS WebSocket 流式、ASR 实时流式
- 视频 `callback_url` 回调加速
- 指标埋点（Prometheus）
- 多租户/API Key 路由

---

## 4. 系统架构

### 4.1 分层
```
┌──────────────────────────────────────────────────────────┐
│ HTTP 层 (handler)   —— 解析 OpenAI 请求、写出 OpenAI 响应 │
├──────────────────────────────────────────────────────────┤
│ 转换层 (translator) —— OpenAI ⇄ 火山 结构互转（纯函数）    │
├──────────────────────────────────────────────────────────┤
│ 客户端层 (provider) —— ark(方舟) / speech(语音) HTTP 调用  │
├──────────────────────────────────────────────────────────┤
│ 基础设施 (config/log/middleware)                          │
└──────────────────────────────────────────────────────────┘
```

### 4.2 设计原则（SOLID / KISS / DRY / YAGNI）
- **S**：handler 仅管 HTTP，translator 仅管结构映射，provider 仅管上游调用。
- **O/D**：handler 依赖 `provider` 接口而非具体实现，便于 mock 测试。
- **DRY**：火山客户端（httpClient/鉴权/重试）共用一份。
- **KISS/YAGNI**：仅引入 1 个外部依赖（yaml.v3），其余用标准库；不预留未使用能力。

---

## 5. 技术选型

| 维度 | 选型 | 理由 |
|---|---|---|
| 语言 | Go 1.23+（`go.mod: go 1.23`） | 用户要求最新版；slog、增强 ServeMux 可用 |
| HTTP 服务 | 标准库 `net/http`（1.22+ ServeMux 方法路由） | 零额外依赖，流式控制直接 |
| HTTP 客户端 | 标准库 `net/http` | 统一，便于流式 io.Copy |
| 日志 | 标准库 `log/slog` | 结构化、零依赖 |
| 配置 | `gopkg.in/yaml.v3` + 环境变量 | 唯一外部依赖 |
| JSON | 标准库 `encoding/json` | 足够 |
| 部署 | Docker 多阶段（builder: golang:1.24-alpine / runtime: alpine） | 镜像小、可复现 |

**外部依赖仅 1 个**：`gopkg.in/yaml.v3`。

---

## 6. 项目目录结构

```
new-api-proxy/
├── cmd/
│   └── proxy/
│       └── main.go                     # 入口：加载配置→启动 server
├── internal/
│   ├── config/
│   │   └── config.go                   # 配置结构 + 加载(env+yaml)
│   ├── model/
│   │   ├── openai.go                   # OpenAI 请求/响应结构
│   │   └── volcano.go                  # 火山 结构
│   ├── provider/
│   │   ├── ark/
│   │   │   └── client.go               # 方舟客户端(chat/embed/image/video)
│   │   └── speech/
│   │       └── client.go               # 语音客户端(tts/asr)
│   ├── translator/
│   │   ├── chat.go                     # 文本：基本透传/模型映射
│   │   ├── image.go                    # 图片：size/n 转换
│   │   ├── video.go                    # 视频：Sora↔Seedance
│   │   ├── tts.go                      # TTS 映射
│   │   └── asr.go                      # ASR 映射
│   ├── handler/
│   │   ├── chat.go                     # /v1/chat/completions
│   │   ├── embeddings.go               # /v1/embeddings
│   │   ├── image.go                    # /v1/images/generations
│   │   ├── video.go                    # /v1/videos, /v1/videos/{id}
│   │   ├── audio_speech.go             # /v1/audio/speech
│   │   ├── audio_transcription.go      # /v1/audio/transcriptions
│   │   └── health.go                   # /healthz, /
│   ├── middleware/
│   │   ├── auth.go                     # API Key 校验
│   │   ├── logger.go                   # 请求日志
│   │   ├── recovery.go                 # panic 恢复
│   │   └── cors.go                     # CORS
│   ├── server/
│   │   ├── server.go                   # server 结构、优雅关停
│   │   └── router.go                   # 路由注册
│   └── pkg/
│       ├── httputil/resp.go            # 统一 JSON 响应/错误
│       └── logx/log.go                 # slog 初始化
├── docs/
│   └── DEVELOPMENT_PLAN.md             # 本文档
├── config.example.yaml                 # 配置示例
├── .env.example                        # 环境变量示例
├── Dockerfile                          # 多阶段构建
├── docker-compose.yml                  # 一键部署
├── .dockerignore
├── .gitignore
├── Makefile                            # build/run/docker/test
├── go.mod / go.sum
└── README.md
```

---

## 7. 核心模块设计

### 7.1 config
```go
type Config struct {
    Server   ServerConfig            // Port, ReadTimeout, WriteTimeout
    Ark      ArkConfig               // APIKey, BaseURL, Region, Timeout
    Speech   SpeechConfig            // AppID, AccessToken, BaseURL(TTS/ASR)
    Proxy    ProxyConfig             // AuthKey(可空=不校验), LogLevel
    ModelMap map[string]string       // OpenAI模型名 -> 火山Model ID
}
```
加载优先级：环境变量 > config.yaml > 默认值。

### 7.2 provider（客户端层）
- `ark.Client`：持有 `*http.Client`、BaseURL、APIKey。方法：
  - `Chat(ctx, body io.Reader) (resp *http.Response, err)` —— 透传，支持 SSE
  - `Embeddings(...)` —— 透传
  - `CreateImage(ctx, req) (*volcano.ImageResponse, err)`
  - `CreateVideoTask(ctx, req) (taskID string, err)`
  - `GetVideoTask(ctx, id) (*volcano.VideoTask, err)`
- `speech.Client`：`Synthesize(ctx, req) (audio []byte, fmt string, err)`、`SubmitASR/QueryASR`。
- 统一：超时、重试（幂等 GET 与 5xx，指数退避）、错误体解析归一为 `*APIError{Code,Message,Status}`。

### 7.3 translator（纯函数转换层）
- 无副作用，入参出参为 model 结构，便于单测。
- 例：`translator.ImageRequest(*openai.ImageRequest, modelMap) (*volcano.ImageRequest, error)`

### 7.4 handler（HTTP 层）
- 解析 OpenAI 请求 → 调 translator → 调 provider → 调反向 translator → 写响应。
- 流式接口（chat）直接 `io.Copy` 上游 `resp.Body` 到 `w`，并设置 `Content-Type: text/event-stream`、`Cache-Control: no-cache`、禁用压缩、`http.Flusher` 逐块刷新。

### 7.5 middleware
- `recovery`：捕获 panic，返回 500 + 结构化日志。
- `logger`：记录 method/path/status/duration/bytes，关联 requestID。
- `auth`：可选校验 `Authorization: Bearer <ProxyAuthKey>`，空配置则放行（内网部署于 new-api 之后）。
- `cors`：允许 new-api 跨域。

---

## 8. 各接口转换规则详述

### 8.1 文本对话（透传）
- 路径：`POST /v1/chat/completions` → `POST {ark}/api/v3/chat/completions`
- 仅做：① `model` 经 `ModelMap` 映射；② 透传 body；③ `stream=true` 时透传 SSE。
- 响应：原样回传（已是 OpenAI 格式）。

### 8.2 嵌入（透传）
- `POST /v1/embeddings` → `POST {ark}/api/v3/embeddings`，模型映射后透传。

### 8.3 文生图（轻转换）
- 路径：`POST /v1/images/generations` → `POST {ark}/api/v3/images/generations`
- 字段映射：

| OpenAI | 火山 | 规则 |
|---|---|---|
| `model` | `model` | ModelMap（dall-e-3/gpt-image-1 → doubao-seedream-*） |
| `prompt` | `prompt` | 直传 |
| `size` | `size` | `1024x1024`→`2K`；`1792x1024`/`1024x1792`→`2K`(16:9/9:16)；自由尺寸透传 |
| `n` (>1) | `sequential_image_generation:"auto"` + `max_images:n` | 火山无 n |
| `response_format` | `response_format` | `url`/`b64_json` 直传 |
| `quality`(gpt-image) | `output_format`/忽略 | 视模型 |
| — | `watermark:false` | 默认关水印 |

- 响应：火山 `data[].url/b64_json` 已近 OpenAI 结构，补 `created` 字段归一。

### 8.4 文生视频（Sora → Seedance，重度转换）
**创建**：`POST /v1/videos` → `POST {ark}/api/v3/contents/generations/tasks`
| OpenAI Sora | 火山 Seedance | 规则 |
|---|---|---|
| `prompt` | `content:[{type:text,text:prompt}]` | 包装为 content 数组 |
| `model` | `model` | sora-2 → doubao-seedance-* |
| `size:"1280x720"` | `resolution:"720p"` + `ratio:"16:9"` | 解析宽高→就近 480p/720p/1080p + 比例 |
| `seconds:"8"` | `duration:8` | string→int |
| `input_reference`(image_url/file_id) | `content:[{type:image_url,image_url:{url},role:first_frame}]` | 首帧 |
| — | `watermark:false`, `generate_audio:true` | 默认值 |

响应：火山 `{id}` → OpenAI `{id, object:"video", status:"queued"}`。

**查询**：`GET /v1/videos/{id}` → `GET {ark}/api/v3/contents/generations/tasks/{id}`
- 状态映射：`queued→queued`、`running→in_progress`、`succeeded→completed`、`failed→failed`、`expired→failed`。
- 成功时 `content.video_url` → OpenAI `output.video_url`（直接给签名下载链接，无需独立 `/content` 端点）。
- 轮询：本代理对「创建」可选内部轮询直到终态再返回（同步体验），或直接返回 task id 由客户端轮询（默认后者，兼容 Sora 语义）。

### 8.5 TTS（重度转换）
- 路径：`POST /v1/audio/speech` → `POST {speech}/api/v1/tts`
- 映射：
| OpenAI | 火山 |
|---|---|
| `model` | 忽略（或映射音色族） |
| `input` | `request.text` |
| `voice` | `audio.voice_type`（经 VoiceMap，默认 `zh_female_cancan_mars_bigtts`） |
| `response_format` | `audio.encoding`（mp3/wav/pcm/ogg） |
| `speed` | `audio.speed_ratio` |
- 补：`app:{appid,token}`、`user:{uid}`、`request:{reqid:uuid, operation:"query"}`。
- 鉴权头：`Authorization: Bearer;{AccessToken}`。
- 响应：火山返回 JSON 含 `data`（Base64 音频）→ 解码为二进制 → 直接写入响应体（`Content-Type: audio/mpeg` 等）。实现 OpenAI「直接返回音频流」语义。

### 8.6 ASR（同步包装异步）
- 路径：`POST /v1/audio/transcriptions` → 火山 ASR 提交+轮询
- 流程：multipart 读音频 → （火山要求 URL：本地暂以 Base64 内联或对象存储 URL；MVP 先支持 `url` 字段直传 + 本地文件转可访问 URL 占位说明）→ 提交任务拿 task_id → 轮询至终态 → 取 `text` → 组装 OpenAI `{text}`。
- 响应：`{ "text": "<转写结果>" }`。

---

## 9. 错误处理与可观测性
- 统一错误结构（OpenAI 风格）：
  ```json
  {"error":{"message":"...","type":"upstream_error","code":"..."}}
  ```
- 上游 4xx/5xx：透传状态码与错误体（尽力解析）。
- 日志：slog JSON，关键字段 `request_id/method/path/status/duration/upstream_status/model`。
- 超时：Ark 默认 120s（视频任务创建快、查询快）；TTS/ASR 单独配置。

---

## 10. 配置设计（config.example.yaml）
```yaml
server:
  port: 8080
  read_timeout: 60s
  write_timeout: 300s      # 视频/ASR 轮询需较长
  log_level: info
proxy:
  auth_key: ""             # 空=不校验（内网）
ark:
  api_key: "${ARK_API_KEY}"
  base_url: "https://ark.cn-beijing.volces.com/api/v3"
  timeout: 120s
speech:
  app_id: "${SPEECH_APP_ID}"
  access_token: "${SPEECH_ACCESS_TOKEN}"
  tts_url: "https://openspeech.bytedance.com/api/v1/tts"
  asr_url: "https://openspeech.bytedance.com/api/v1/auc"
  timeout: 60s
model_map:
  "gpt-4o": "doubao-seed-1-6-250615"
  "gpt-4o-mini": "doubao-1-5-lite-32k-250115"
  "text-embedding-3-small": "doubao-embedding-text-240715"
  "dall-e-3": "doubao-seedream-3-0-t2i-250415"
  "sora-2": "doubao-seedance-1-0-pro-250528"
voice_map:
  "alloy": "zh_female_cancan_mars_bigtts"
  "nova": "zh_male_M392_conversation_wvae"
```
环境变量同名覆盖（`SERVER_PORT`、`ARK_API_KEY` 等）。

---

## 11. Docker 部署
- **Dockerfile**（多阶段）：
  - `builder`：`golang:1.24-alpine`，`go mod download` → `CGO_ENABLED=0 go build -ldflags "-s -w"` 产出静态二进制。
  - `runtime`：`alpine:3.20`（带 ca-certificates），复制二进制，`EXPOSE 8080`，非 root 用户运行。
- **docker-compose.yml**：挂载 `config.yaml`、注入环境变量、健康检查、端口映射。
- 镜像目标 < 30MB。

---

## 12. 与 new-api 对接说明
在 new-api「渠道」中新增：
- 类型：**OpenAI**
- Base URL：`http://new-api-proxy:8080`
- 密钥：本代理 `proxy.auth_key`（若未启用校验则任意）
- 模型：填 OpenAI 别名（如 `gpt-4o`、`dall-e-3`、`sora-2`），由本代理 `model_map` 解析为火山模型。

---

## 13. 开发与测试策略
- 单元测试聚焦 **translator 纯函数**（无网络依赖，覆盖各字段映射与边界）。
- provider/handler 通过接口注入 mock，保证可测。
- 验证：`go build ./...`、`go vet ./...`、Docker 构建 + 启动健康检查。
- （API Key 缺失时）仅做编译与启动级验证，不发起真实上游调用。

---

## 14. 任务里程碑（与 TaskList 对应）
1. 初始化骨架（go.mod/目录/Makefile） ✅
2. 配置模块
3. 数据模型
4. 方舟客户端
5. 语音客户端
6. 中间件
7. Chat/Embeddings 透传
8. 文生图
9. 文生视频（Sora→Seedance）
10. TTS
11. ASR
12. 路由与启动
13. Docker 部署
14. README/配置示例
15. 构建验证
16. 提交代码
