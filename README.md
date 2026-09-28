[简体中文](README.md) | [English](README.en.md)

# new-api-proxy

OpenAI ⇄ 火山引擎格式转换代理。用 Go 实现，单二进制 + Docker 部署。

![Go](https://img.shields.io/badge/Go-1.23%2B-00ADD8) ![Docker](https://img.shields.io/badge/deploy-Docker-2496ED) ![License](https://img.shields.io/badge/license-MIT-blue)

## 项目介绍

将 **OpenAI / Sora 格式**的请求，转换为**火山引擎（火山方舟 Ark + 语音技术 OpenSpeech）**支持的参数，覆盖**文本、向量、图片、视频、语音合成（TTS）、语音识别（ASR）**六大能力域。

本服务对外是标准 OpenAI 兼容入口，专为对接 **new-api / one-api** 等网关设计——在 new-api 里把它当成一个「OpenAI 渠道」即可直接接入，无需改造，也无需在网关侧维护火山 Model ID。链路：`客户端 → new-api（OpenAI 格式）→ 本代理（转换）→ 火山引擎（Ark / OpenSpeech）`，内部按 `handler → translator（纯函数互转）→ provider（ark/speech HTTP 客户端）` 单向分层。

## ✨ 功能特性

- **OpenAI 兼容入口**：new-api 把本服务当成一个 OpenAI 渠道即可，无需改造
- **六大能力域转换**：文本对话、向量嵌入、文生图、文生视频、TTS、ASR
- **流式支持**：Chat completions 的 SSE 透传
- **模型/音色映射**：OpenAI 模型名 → 火山 Model ID，OpenAI voice → 火山 voice_type
- **轻量依赖**：仅 1 个外部依赖（`yaml.v3`），其余全部标准库
- **生产就绪**：结构化日志（slog）、统一错误归一、优雅关停、健康检查、非 root 容器

### 转换矩阵

| 能力 | OpenAI 入口 | 火山目标 | 转换 |
|---|---|---|---|
| 文本对话 | `POST /v1/chat/completions` | `POST /api/v3/chat/completions` | 透传 + SSE |
| 向量嵌入 | `POST /v1/embeddings` | `POST /api/v3/embeddings` | 透传 |
| 文生图 | `POST /v1/images/generations` | `POST /api/v3/images/generations` | 轻转换（size/n） |
| 文生视频 | `POST /v1/videos` · `GET /v1/videos/{id}` | `POST /api/v3/contents/generations/tasks` · `GET .../{id}` | 重度转换（Sora→Seedance） |
| 语音合成 | `POST /v1/audio/speech` | `POST openspeech.bytedance.com/api/v1/tts` | 重度转换 |
| 语音识别 | `POST /v1/audio/transcriptions` | 火山 ASR 异步任务 | 同步包装异步 |

> ⚠️ TTS/ASR 属火山「语音技术」独立产品线，鉴权与方舟不同（`appid` + `access_token`，Header `Bearer;{token}`），需单独开通。

## 🛠 技术栈

Go 1.23（CGO 禁用，静态二进制），唯一第三方依赖 `gopkg.in/yaml.v3`；多阶段 Dockerfile 构建，docker-compose 一键部署。

## 🚀 快速开始

1. 准备配置与密钥：`cp config.example.yaml config.yaml`、`cp .env.example .env`，编辑 `.env` 填入 `ARK_API_KEY / SPEECH_APP_ID / SPEECH_ACCESS_TOKEN`，编辑 `config.yaml` 调整 `model_map / voice_map`。
2. 启动并验证：

   ```bash
   docker compose up -d --build
   curl http://localhost:8080/healthz   # {"status":"ok"}
   ```

### 对接 new-api

在 new-api「渠道」中新增：类型选 **OpenAI**，Base URL 填 `http://new-api-proxy:8080`（或宿主 `http://host:8080`），密钥填本代理 `proxy.auth_key`（未启用校验则任意），模型填 OpenAI 别名（如 `gpt-4o`、`dall-e-3`、`sora-2`、`tts-1`、`whisper-1`），模型名经 `model_map` 解析为火山模型。

### 配置

`config.yaml` 同名大写环境变量优先级更高（如 `ARK_API_KEY`），完整示例见 [`config.example.yaml`](config.example.yaml)：

```yaml
server:
  port: 8080
  write_timeout: 300s     # 视频/ASR 轮询需较长
proxy:
  auth_key: ""            # 空=不校验入口
ark:
  api_key: "${ARK_API_KEY}"
  base_url: "https://ark.cn-beijing.volces.com/api/v3"
speech:
  app_id: "${SPEECH_APP_ID}"
  access_token: "${SPEECH_ACCESS_TOKEN}"
model_map:
  "gpt-4o": "doubao-seed-1-6-250615"
  "sora-2": "doubao-seedance-1-0-pro-250528"
voice_map:
  "alloy": "zh_female_cancan_mars_bigtts"
```

### 接口示例

**文生视频（Sora 格式，异步任务）**

```bash
# 创建
curl -X POST http://localhost:8080/v1/videos -H "Content-Type: application/json" \
  -d '{"model":"sora-2","prompt":"海浪拍打沙滩","size":"1280x720","seconds":"5"}'
# {"id":"cgt-...","object":"video","status":"queued"}

# 轮询
curl http://localhost:8080/v1/videos/cgt-xxxx
# {"id":"cgt-xxxx","object":"video","status":"completed","output":{"video_url":"https://..."}}
```

### 本地开发

需要 **Go 1.23+**：`make tidy`（整理依赖）、`make run`（直接运行）、`make build`（产出 `bin/new-api-proxy`）、`make vet`（静态检查）、`make test`（单元测试）、`make docker`（构建+运行镜像）。

## 📁 目录结构

```
cmd/proxy/                 入口 main
internal/
  config/ model/           配置加载；OpenAI / 火山数据结构与透传工具
  provider/{ark,speech}/   上游 HTTP 客户端
  translator/              结构互转（纯函数）
  handler/ middleware/     OpenAI 入口处理器；auth/logger/recovery/cors/requestid
  server/ pkg/             路由与生命周期；响应/日志工具
Dockerfile / docker-compose.yml / docs/DEVELOPMENT_PLAN.md
```

## ⚠️ 注意事项

1. **TTS/ASR 独立产品线**：需在火山控制台单独开通语音技术，获取 `appid` 与 `access_token`，与方舟 API Key 体系不同。
2. **ASR 需音频 URL**：火山录音文件识别要求可公网访问的音频 URL，请通过 multipart 字段 `url` 提供（当前 MVP 不做本地文件→对象存储转存）。
3. **视频为异步任务**：创建返回 task id，需轮询 `GET /v1/videos/{id}` 获取结果（响应内含签名下载链接 `output.video_url`，约 24h 有效）。
4. **流式超时**：长流式/视频场景请调大 `server.write_timeout` 或设为 `0`（不限）。
5. **OpenAI Sora 上游变化**：OpenAI Videos API 计划停用，本代理保留转换能力供火山侧使用。

## 📄 License

MIT
