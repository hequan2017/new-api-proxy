# new-api-proxy Makefile
# 约定：所有命令均为幂等、可重复执行。

APP        := new-api-proxy
PKG        := github.com/hequan2017/new-api-proxy
BIN_DIR    := bin
MAIN       := $(PKG)/cmd/proxy
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo unknown)
LDFLAGS    := -s -w -X $(PKG)/internal/version.Version=$(VERSION) -X $(PKG)/internal/version.Commit=$(COMMIT) -X $(PKG)/internal/version.BuildTime=$(BUILD_TIME)

DOCKER_IMAGE := $(APP):latest

.PHONY: all tidy build run fmt vet test docker build-docker clean help

all: build

## 依赖整理
tidy:
	go mod tidy

## 本地构建（静态二进制）
build:
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP) $(MAIN)

## 直接运行（需 config.yaml 或环境变量）
run:
	go run $(MAIN)

## 格式化
fmt:
	gofmt -s -w .

## 静态检查
vet:
	go vet ./...

## 单元测试
test:
	go test ./... -cover

## 构建并运行本地镜像
docker: build-docker
	docker run --rm -p 8080:8080 --env-file .env -v $(PWD)/config.yaml:/app/config.yaml $(DOCKER_IMAGE)

## 构建 Docker 镜像
build-docker:
	docker build -t $(DOCKER_IMAGE) .

## 清理产物
clean:
	rm -rf $(BIN_DIR) $(DIST_DIR)

help:
	@echo "new-api-proxy 构建命令："
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //'
