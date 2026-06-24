# syntax=docker/dockerfile:1

# ===== Builder：编译静态二进制 =====
FROM golang:1.24-alpine AS builder

WORKDIR /build

# 仅拷依赖清单，利用层缓存；go.sum 不存在时由 go mod download 生成。
COPY go.mod go.sum* ./
RUN go mod download

# 拷源码
COPY . .

# 版本信息（构建参数）
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_TIME=unknown

# 静态构建（CGO 关闭，便于 scratch/alpine 运行）
RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags "-s -w \
          -X github.com/hequan2017/new-api-proxy/internal/version.Version=${VERSION} \
          -X github.com/hequan2017/new-api-proxy/internal/version.Commit=${COMMIT} \
          -X github.com/hequan2017/new-api-proxy/internal/version.BuildTime=${BUILD_TIME}" \
        -o /out/new-api-proxy \
        ./cmd/proxy

# ===== Runtime：最小运行镜像 =====
FROM alpine:3.20

# ca-certificates：HTTPS 上游必需；tzdata：时区；非 root 用户。
RUN apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -u 10001 app

WORKDIR /app
COPY --from=builder /out/new-api-proxy /app/new-api-proxy

USER app
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/healthz >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/app/new-api-proxy"]
CMD ["-config", "/app/config.yaml"]
