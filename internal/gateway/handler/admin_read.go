package handler

import (
	"net/http"

	"github.com/Jayleonc/ai-gateway/internal/metering"
	"github.com/Jayleonc/ai-gateway/internal/policy/quota"
	"github.com/gin-gonic/gin"
)

type AdminReadHandler struct {
	quotaStore  quota.QuotaStore
	usageQuery  metering.UsageQuery
	meteringRec metering.Recorder
}

func NewAdminReadHandler(qs quota.QuotaStore, uq metering.UsageQuery, rec metering.Recorder) *AdminReadHandler {
	return &AdminReadHandler{
		quotaStore:  qs,
		usageQuery:  uq,
		meteringRec: rec,
	}
}

type QuotaFactView struct {
	APIKeyID  string `json:"api_key_id"`
	Used      int64  `json:"used"`
	Remaining int64  `json:"remaining"`
	Total     int64  `json:"total"`
}

type UsageFactView struct {
	RequestID       string `json:"request_id"`
	Model           string `json:"model"`
	ConfirmedTokens int64  `json:"confirmed_tokens"`
	Status          string `json:"status"`
	EndReason       string `json:"end_reason"`
}

type GovernanceSummaryFactView struct {
	TotalKeys     int `json:"total_keys"`
	ActiveKeys    int `json:"active_keys"`
	LimitedKeys   int `json:"limited_keys"`
	ExhaustedKeys int `json:"exhausted_keys"`
}

// GetQuotaFact returns the quota fact for a given API key from Gateway's in-memory store
func (h *AdminReadHandler) GetQuotaFact(c *gin.Context) {
	if h == nil || h.quotaStore == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "quota store not available"})
		return
	}

	apiKeyID := c.Param("api_key_id")
	if apiKeyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "api_key_id is required"})
		return
	}

	remaining := h.quotaStore.Remaining(apiKeyID)

	var used int64
	if h.usageQuery != nil {
		recs, err := h.usageQuery.ListByAPIKey(apiKeyID, 1000000)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for _, r := range recs {
			if r != nil {
				used += r.ConfirmedTokens
			}
		}
	}

	c.JSON(http.StatusOK, &QuotaFactView{
		APIKeyID:  apiKeyID,
		Used:      used,
		Remaining: remaining,
		Total:     used + remaining,
	})
}

// GetUsagesFact returns usage records for a given API key from Gateway's in-memory recorder
func (h *AdminReadHandler) GetUsagesFact(c *gin.Context) {
	if h == nil || h.usageQuery == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "usage query not available"})
		return
	}

	apiKeyID := c.Param("api_key_id")
	if apiKeyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "api_key_id is required"})
		return
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := parseIntParam(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	records, err := h.usageQuery.ListByAPIKey(apiKeyID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	usages := make([]UsageFactView, 0, len(records))
	for _, r := range records {
		if r != nil {
			usages = append(usages, UsageFactView{
				RequestID:       r.RequestID,
				Model:           r.Model,
				ConfirmedTokens: r.ConfirmedTokens,
				Status:          r.Status,
				EndReason:       r.EndReason,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"api_key_id": apiKeyID,
		"usages":     usages,
		"total":      len(usages),
	})
}

func parseIntParam(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
