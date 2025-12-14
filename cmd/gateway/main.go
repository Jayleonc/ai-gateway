package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Jayleonc/ai-gateway/internal/gateway"
	"github.com/Jayleonc/ai-gateway/internal/identity"
	"github.com/Jayleonc/ai-gateway/internal/policy"
	"github.com/Jayleonc/ai-gateway/internal/policy/quota"
	"github.com/Jayleonc/ai-gateway/internal/policy/ratelimit"
	"github.com/Jayleonc/ai-gateway/internal/policy/routing"
	"github.com/Jayleonc/ai-gateway/internal/provider"
	"github.com/Jayleonc/ai-gateway/internal/provider/openai"
)

func main() {
	log.Println("Starting AI Gateway...")

	// ========================================
	// 依赖组装（手动 wire up）
	// ========================================

	// 1. Identity 模块
	apiKeyRepo := identity.NewAPIKeyRepository()
	authenticator := identity.NewAuthenticator(apiKeyRepo)

	// 2. Policy 模块
	quotaStore := quota.NewInMemoryStore()
	quotaStore.SetRemaining("gw-key-001", 1)
	quotaChecker := quota.NewChecker(quotaStore)
	rateLimiter := ratelimit.NewLimiter()
	router := routing.NewRouter()
	policyEngine := policy.NewEngine(quotaChecker, rateLimiter, router)

	// 3. Provider 模块
	providerRegistry := provider.NewRegistry()

	// 注册 OpenAI Provider
	openaiAdapter := openai.NewAdapter("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", "") // API Key 从配置读取
	providerRegistry.Register(openaiAdapter)

	// 4. Gateway 模块
	routerCfg := &gateway.RouterConfig{
		Authenticator:    authenticator,
		PolicyEngine:     policyEngine,
		ProviderRegistry: providerRegistry,
	}
	ginRouter := gateway.SetupRouter(routerCfg)

	serverCfg := &gateway.ServerConfig{
		Host: "0.0.0.0",
		Port: 8520,
	}
	server := gateway.NewServer(serverCfg, ginRouter)

	// ========================================
	// 启动服务
	// ========================================

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	log.Println("AI Gateway is running on :8520")
	log.Println("Health check: http://localhost:8520/health")
	log.Println("OpenAI API: http://localhost:8520/v1/chat/completions")

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("AI Gateway stopped")
}
