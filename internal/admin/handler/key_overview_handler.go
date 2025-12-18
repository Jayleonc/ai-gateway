package handler

import (
	"net/http"
	"strconv"

	"github.com/Jayleonc/ai-gateway/internal/admin/service"
	"github.com/gin-gonic/gin"
)

type GovernanceHandler struct {
	keyViewService    *service.KeyViewService
	governanceService *service.GovernanceService
}

func NewGovernanceHandler(
	keyViewService *service.KeyViewService,
	governanceService *service.GovernanceService,
) *GovernanceHandler {
	return &GovernanceHandler{
		keyViewService:    keyViewService,
		governanceService: governanceService,
	}
}

func (h *GovernanceHandler) GetOverview(c *gin.Context) {
	apiKeyID := c.Param("api_key_id")
	if apiKeyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "api_key_id is required"})
		return
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	overview, err := h.keyViewService.GetOverview(c.Request.Context(), apiKeyID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, overview)
}

func (h *GovernanceHandler) ListUsages(c *gin.Context) {
	apiKeyID := c.Param("api_key_id")
	if apiKeyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "api_key_id is required"})
		return
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	usages, err := h.keyViewService.ListUsages(c.Request.Context(), apiKeyID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, usages)
}

func (h *GovernanceHandler) GetGovernance(c *gin.Context) {
	apiKeyID := c.Param("api_key_id")
	if apiKeyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "api_key_id is required"})
		return
	}

	governance, err := h.keyViewService.GetGovernance(c.Request.Context(), apiKeyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, governance)
}

func (h *GovernanceHandler) ResetQuota(c *gin.Context) {
	apiKeyID := c.Param("api_key_id")
	if apiKeyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "api_key_id is required"})
		return
	}

	result, err := h.keyViewService.ResetQuota(c.Request.Context(), apiKeyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *GovernanceHandler) GetSummary(c *gin.Context) {
	summary, err := h.governanceService.GetSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}
