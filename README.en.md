[简体中文](README.md) | [English](README.en.md)

# new-api-proxy

An OpenAI ⇄ Volcano Engine format conversion proxy, implemented in Go and deployed as a single binary or via Docker.

![Go](https://img.shields.io/badge/Go-1.23%2B-00ADD8) ![Docker](https://img.shields.io/badge/deploy-Docker-2496ED) ![License](https://img.shields.io/badge/license-MIT-blue)

## Introduction

This service converts requests in **OpenAI / Sora format** into parameters supported by **Volcano Engine (Ark + OpenSpeech)**, covering six capability domains: **text, embeddings, images, video, text-to-speech (TTS), and speech recognition (ASR)**.

Externally it exposes a standard OpenAI-compatible endpoint, designed to plug into gateways such as **new-api / one-api** — just add it in new-api as an "OpenAI channel" with no gateway-side changes and no need to maintain Volcano Model IDs there. Flow: `client → new-api (OpenAI format) → this proxy (conversion) → Volcano Engine (Ark / OpenSpeech)`, layered one-way internally as `handler → translator (pure-function conversion) → provider (ark/speech HTTP clients)`.

## ✨ Features

- **OpenAI-compatible endpoint**: new-api treats this service as a plain OpenAI channel, zero rework
- **Six capability domains**: chat, embeddings, text-to-image, text-to-video, TTS, ASR
- **Streaming**: SSE pass-through for chat completions
- **Model/voice mapping**: OpenAI model names → Volcano Model IDs, OpenAI voices → Volcano voice_types
- **Lightweight**: a single external dependency (`yaml.v3`), everything else is the standard library
- **Production ready**: structured logging (slog), unified error normalization, graceful shutdown, health check, non-root container

### Conversion Matrix

| Capability | OpenAI Endpoint | Volcano Target | Conversion |
|---|---|---|---|
| Chat | `POST /v1/chat/completions` | `POST /api/v3/chat/completions` | Pass-through + SSE |
| Embeddings | `POST /v1/embeddings` | `POST /api/v3/embeddings` | Pass-through |
| Text-to-image | `POST /v1/images/generations` | `POST /api/v3/images/generations` | Light (size/n) |
| Text-to-video | `POST /v1/videos` · `GET /v1/videos/{id}` | `POST /api/v3/contents/generations/tasks` · `GET .../{id}` | Heavy (Sora→Seedance) |
| TTS | `POST /v1/audio/speech` | `POST openspeech.bytedance.com/api/v1/tts` | Heavy |
| ASR | `POST /v1/audio/transcriptions` | Volcano async ASR tasks | Sync wrapper over async |

> ⚠️ TTS/ASR belong to Volcano's standalone "Speech Technology" product line. Authentication differs from Ark (`appid` + `access_token`, header `Bearer;{token}`) and must be enabled separately.

## 🛠 Tech Stack

Go 1.23 (CGO disabled, static binary) with `gopkg.in/yaml.v3` as the only third-party dependency; multi-stage Dockerfile build with one-command docker-compose deployment.

## 🚀 Quick Start

1. Prepare config and secrets: `cp config.example.yaml config.yaml` and `cp .env.example .env`; edit `.env` to fill in `ARK_API_KEY / SPEECH_APP_ID / SPEECH_ACCESS_TOKEN`, and edit `config.yaml` to adjust `model_map / voice_map`.
2. Start and verify:

   ```bash
   docker compose up -d --build
   curl http://localhost:8080/healthz   # {"status":"ok"}
   ```

### Integrating with new-api

Add a channel in new-api: type **OpenAI**, Base URL `http://new-api-proxy:8080` (or `http://host:8080` from the host), key = this proxy's `proxy.auth_key` (anything if auth is disabled), models = OpenAI aliases (e.g. `gpt-4o`, `dall-e-3`, `sora-2`, `tts-1`, `whisper-1`); names are resolved to Volcano models via `model_map`.

### Configuration

Same-named uppercase environment variables in `config.yaml` take precedence (e.g. `ARK_API_KEY`); see [`config.example.yaml`](config.example.yaml) for the full example:

```yaml
server:
  port: 8080
  write_timeout: 300s     # video/ASR polling needs a longer window
proxy:
  auth_key: ""            # empty = no inbound auth
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

### API Examples

**Text-to-video (Sora format, async task)**

```bash
# Create
curl -X POST http://localhost:8080/v1/videos -H "Content-Type: application/json" \
  -d '{"model":"sora-2","prompt":"waves on the beach","size":"1280x720","seconds":"5"}'
# {"id":"cgt-...","object":"video","status":"queued"}

# Poll
curl http://localhost:8080/v1/videos/cgt-xxxx
# {"id":"cgt-xxxx","object":"video","status":"completed","output":{"video_url":"https://..."}}
```

### Local Development

Requires **Go 1.23+**: `make tidy` (tidy deps), `make run` (run directly), `make build` (produce `bin/new-api-proxy`), `make vet` (static checks), `make test` (unit tests), `make docker` (build and run the image).

## 📁 Project Structure

```
cmd/proxy/                 main entrypoint
internal/
  config/ model/           config loading; OpenAI / Volcano data structures and pass-through helpers
  provider/{ark,speech}/   upstream HTTP clients
  translator/              struct conversion (pure functions)
  handler/ middleware/     OpenAI endpoint handlers; auth/logger/recovery/cors/requestid
  server/ pkg/             routing and lifecycle; response/log utilities
Dockerfile / docker-compose.yml / docs/DEVELOPMENT_PLAN.md
```

## ⚠️ Notes

1. **TTS/ASR are a standalone product line**: enable Volcano Speech Technology in its console first and obtain `appid` and `access_token`; they are separate from the Ark API Key system.
2. **ASR requires an audio URL**: Volcano file recognition needs a publicly reachable audio URL, provided via the multipart field `url` (the current MVP does not upload local files to object storage).
3. **Video is an async task**: creation returns a task id; poll `GET /v1/videos/{id}` for the result (the response carries a signed download link `output.video_url`, valid for about 24h).
4. **Streaming timeouts**: for long streams or video, raise `server.write_timeout` or set it to `0` (unlimited).
5. **OpenAI Sora upstream changes**: the OpenAI Videos API is scheduled for deprecation; this proxy keeps the conversion capability for the Volcano side.

## 📄 License

MIT
