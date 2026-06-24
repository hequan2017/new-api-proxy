// Command new-api-proxy 是 OpenAI ⇄ 火山引擎 的格式转换代理服务。
//
// 用法：
//
//	new-api-proxy -config config.yaml
//
// 配置优先级：环境变量 > config.yaml > 默认值。
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hequan2017/new-api-proxy/internal/config"
	"github.com/hequan2017/new-api-proxy/internal/handler"
	"github.com/hequan2017/new-api-proxy/internal/pkg/logx"
	"github.com/hequan2017/new-api-proxy/internal/provider/ark"
	"github.com/hequan2017/new-api-proxy/internal/provider/speech"
	"github.com/hequan2017/new-api-proxy/internal/server"
	"github.com/hequan2017/new-api-proxy/internal/version"
)

func main() {
	configPath := flag.String("config", "config.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("加载配置失败", "error", err, "path", *configPath)
		os.Exit(1)
	}

	log := logx.Init(cfg.LogLevel())
	log.Info("启动 new-api-proxy", "version", version.String(), "config", *configPath)

	// 启动期提示未配置的能力域（不阻断，按需报错由各 handler 兜底）。
	if cfg.Ark.APIKey == "" {
		log.Warn("ARK API Key 未配置，文本/嵌入/图片/视频接口将不可用")
	}
	if cfg.Speech.AppID == "" || cfg.Speech.AccessToken == "" {
		log.Warn("语音技术凭证未配置，TTS/ASR 接口将不可用")
	}

	deps := &handler.Deps{
		Cfg: cfg,
		Ark: ark.New(cfg.Ark.BaseURL, cfg.Ark.APIKey),
		Speech: speech.New(
			cfg.Speech.AppID, cfg.Speech.AccessToken,
			cfg.Speech.TTSURL, cfg.Speech.ASRSubmitURL, cfg.Speech.ASRQueryURL,
		),
		Log: log,
	}

	srv := server.New(deps)

	// 异步启动，主线程等待退出信号以优雅关停。
	errCh := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		log.Error("服务异常退出", "error", err)
		os.Exit(1)
	case sig := <-stop:
		log.Info("收到退出信号", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("优雅关停失败", "error", err)
		os.Exit(1)
	}
	log.Info("服务已停止")
}
