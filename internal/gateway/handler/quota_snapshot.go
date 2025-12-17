package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Jayleonc/ai-gateway/internal/metering"
	"github.com/Jayleonc/ai-gateway/internal/policy/quota"
)

type QuotaHandler struct {
	quotaStore quota.QuotaStore
	usageQuery metering.UsageQuery
}

func NewQuotaHandler(qs quota.QuotaStore, uq metering.UsageQuery) *QuotaHandler {
	return &QuotaHandler{quotaStore: qs, usageQuery: uq}
}

type QuotaSnapshotView struct {
	QuotaTotal     int64      `json:"quota_total"`
	QuotaUsed      int64      `json:"quota_used"`
	QuotaRemaining int64      `json:"quota_remaining"`
	LastUpdatedAt  *time.Time `json:"last_updated_at,omitempty"`
}

func (h *QuotaHandler) GetSnapshot(c *gin.Context) {
	if h == nil || h.quotaStore == nil {
		c.Status(http.StatusNotFound)
		return
	}

	apiKeyID := c.Param("api_key")
	if apiKeyID == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	remaining := h.quotaStore.Remaining(apiKeyID)

	var used int64
	var lastUpdated *time.Time
	if h.usageQuery != nil {
		recs, err := h.usageQuery.ListByAPIKey(apiKeyID, 1000000)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for _, r := range recs {
			if r == nil {
				continue
			}
			used += r.ConfirmedTokens

			candidate := r.CompletedAt
			if candidate.IsZero() {
				candidate = r.RequestedAt
			}
			if candidate.IsZero() {
				continue
			}
			if lastUpdated == nil || candidate.After(*lastUpdated) {
				t := candidate
				lastUpdated = &t
			}
		}
	}

	c.JSON(http.StatusOK, &QuotaSnapshotView{
		QuotaTotal:     used + remaining,
		QuotaUsed:      used,
		QuotaRemaining: remaining,
		LastUpdatedAt:  lastUpdated,
	})
}
