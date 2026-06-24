package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/hequan2017/new-api-proxy/internal/handler"
)

// Server 封装 http.Server 与生命周期。
type Server struct {
	httpServer *http.Server
	log        *slog.Logger
}

// New 依据依赖组装路由并构造 Server。
func New(deps *handler.Deps) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:              fmt.Sprintf(":%d", deps.Cfg.Server.Port),
			Handler:           buildHandler(deps),
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       deps.Cfg.Server.ReadTimeout,
			WriteTimeout:      deps.Cfg.Server.WriteTimeout,
			IdleTimeout:       120 * time.Second,
		},
		log: deps.Log,
	}
}

// Start 监听并服务；返回 http.ErrServerClosed 时不视为错误。
func (s *Server) Start() error {
	s.log.Info("http server listening", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown 优雅关停。
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("http server shutting down")
	return s.httpServer.Shutdown(ctx)
}
