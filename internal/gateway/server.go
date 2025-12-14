package gateway

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Server HTTP 服务器
type Server struct {
	httpServer *http.Server
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string
	Port int
}

// NewServer 创建服务器
func NewServer(cfg *ServerConfig, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			Handler:      handler,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 120 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	log.Printf("Starting server on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown 优雅关闭
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")
	return s.httpServer.Shutdown(ctx)
}
