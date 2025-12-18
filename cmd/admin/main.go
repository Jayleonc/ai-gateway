package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Jayleonc/ai-gateway/internal/admin/gatewayclient"
	adminhandler "github.com/Jayleonc/ai-gateway/internal/admin/handler"
	"github.com/Jayleonc/ai-gateway/internal/admin/service"
	"github.com/Jayleonc/ai-gateway/internal/identity"
	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("Starting Admin Server...")

	// ========================================
	// 依赖组装（Phase 7.2 Read-Model Bridging）
	// ========================================

	// 1. Identity
	apiKeyRepo := identity.NewAPIKeyRepository()
	knownKeyIDs := []string{"gw-key-001", "gw-key-002"}

	// 2. Gateway Admin Client (read-only bridge to Gateway facts)
	gatewayURL := "http://localhost:8520"
	gatewayAdminClient := gatewayclient.NewGatewayAdminClient(gatewayURL)

	// ========================================
	// Admin Service & Handler
	// ========================================

	keyViewService := service.NewKeyViewService(apiKeyRepo, gatewayAdminClient)
	governanceService := service.NewGovernanceService(apiKeyRepo, gatewayAdminClient, knownKeyIDs)
	governanceHandler := adminhandler.NewGovernanceHandler(keyViewService, governanceService)

	// ========================================
	// Gin Router - 5 Governance Endpoints
	// ========================================

	r := gin.Default()

	// 1️⃣ Key Overview（单 Key 治理视角）
	r.GET("/admin/keys/:api_key_id/overview", governanceHandler.GetOverview)

	// 2️⃣ Key Usage List（治理证据视角）
	r.GET("/admin/keys/:api_key_id/usages", governanceHandler.ListUsages)

	// 3️⃣ Key Governance State（轻量状态接口）
	r.GET("/admin/keys/:api_key_id/governance", governanceHandler.GetGovernance)

	// 4️⃣ Reset Quota（治理动作，唯一写接口）
	r.POST("/admin/keys/:api_key_id/reset-quota", governanceHandler.ResetQuota)

	// 5️⃣ Governance Summary（平台级治理总览）
	r.GET("/admin/governance/summary", governanceHandler.GetSummary)

	// ========================================
	// HTTP Server
	// ========================================

	srv := &http.Server{
		Addr:    ":8521",
		Handler: r,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("Admin Server is running on :8521")
		log.Println("Phase 6 Admin Governance v1 - 5 Endpoints:")
		log.Println("  1. GET  /admin/keys/:api_key_id/overview")
		log.Println("  2. GET  /admin/keys/:api_key_id/usages")
		log.Println("  3. GET  /admin/keys/:api_key_id/governance")
		log.Println("  4. POST /admin/keys/:api_key_id/reset-quota")
		log.Println("  5. GET  /admin/governance/summary")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Admin Server stopped")
}
