package gateway

import (
	"github.com/gin-gonic/gin"

	"github.com/Jayleonc/ai-gateway/internal/gateway/handler"
	"github.com/Jayleonc/ai-gateway/internal/gateway/middleware"
	"github.com/Jayleonc/ai-gateway/internal/identity"
	"github.com/Jayleonc/ai-gateway/internal/policy"
	"github.com/Jayleonc/ai-gateway/internal/provider"
)

// RouterConfig 路由配置
type RouterConfig struct {
	Authenticator    identity.Authenticator
	PolicyEngine     policy.Engine
	ProviderRegistry provider.Registry
}

// SetupRouter 配置路由
func SetupRouter(cfg *RouterConfig) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	// 全局中间件
	r.Use(middleware.Recovery())
	r.Use(middleware.RequestID())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// OpenAI Compatible API
	v1 := r.Group("/v1")
	{
		// Chat Completions
		chatHandler := handler.NewChatHandler(cfg.Authenticator, cfg.PolicyEngine, cfg.ProviderRegistry)
		v1.POST("/chat/completions", chatHandler.Handle)

		// Completions
		completionsHandler := handler.NewCompletionsHandler(cfg.Authenticator, cfg.PolicyEngine, cfg.ProviderRegistry)
		v1.POST("/completions", completionsHandler.Handle)

		// Models
		modelsHandler := handler.NewModelsHandler(cfg.ProviderRegistry)
		v1.GET("/models", modelsHandler.List)
		v1.GET("/models/:model", modelsHandler.Get)
	}

	return r
}
