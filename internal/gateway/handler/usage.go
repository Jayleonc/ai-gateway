package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Jayleonc/ai-gateway/internal/metering"
)

type UsageHandler struct {
	query metering.UsageQuery
}

func NewUsageHandler(q metering.UsageQuery) *UsageHandler {
	return &UsageHandler{query: q}
}

type UsageView struct {
	RequestID       string     `json:"request_id"`
	APIKeyID        string     `json:"api_key_id"`
	Provider        string     `json:"provider"`
	Model           string     `json:"model"`
	Status          string     `json:"status"`
	EndReason       string     `json:"end_reason"`
	ConfirmedTokens int64      `json:"confirmed_tokens"`
	ChunkCount      int        `json:"chunk_count"`
	StartAt         time.Time  `json:"start_at"`
	FirstChunkAt    *time.Time `json:"first_chunk_at,omitempty"`
	EndAt           *time.Time `json:"end_at,omitempty"`
	Duration        int64      `json:"duration"`
	ErrorMsg        string     `json:"error_msg,omitempty"`
}

func toUsageView(r *metering.UsageRecord) *UsageView {
	if r == nil {
		return nil
	}

	duration := int64(0)
	if r.Duration != 0 {
		duration = r.Duration.Milliseconds()
	} else if r.EndAt != nil {
		duration = r.EndAt.Sub(r.StartAt).Milliseconds()
	}

	return &UsageView{
		RequestID:       r.RequestID,
		APIKeyID:        r.APIKeyID,
		Provider:        r.Provider,
		Model:           r.Model,
		Status:          r.Status,
		EndReason:       r.EndReason,
		ConfirmedTokens: r.ConfirmedTokens,
		ChunkCount:      r.ChunkCount,
		StartAt:         r.StartAt,
		FirstChunkAt:    r.FirstChunkAt,
		EndAt:           r.EndAt,
		Duration:        duration,
		ErrorMsg:        r.ErrorMsg,
	}
}

func (h *UsageHandler) List(c *gin.Context) {
	if h == nil || h.query == nil {
		c.JSON(http.StatusOK, []UsageView{})
		return
	}

	apiKeyID := c.Query("api_key_id")
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	records, err := h.query.ListByAPIKey(apiKeyID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	out := make([]*UsageView, 0, len(records))
	for _, r := range records {
		if v := toUsageView(r); v != nil {
			out = append(out, v)
		}
	}
	c.JSON(http.StatusOK, out)
}

func (h *UsageHandler) Get(c *gin.Context) {
	if h == nil || h.query == nil {
		c.Status(http.StatusNotFound)
		return
	}

	requestID := c.Param("request_id")
	if requestID == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	rec, err := h.query.GetByRequestID(requestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rec == nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, toUsageView(rec))
}
