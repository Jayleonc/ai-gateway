package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	coreerrors "github.com/Jayleonc/ai-gateway/internal/core/errors"
	"github.com/Jayleonc/ai-gateway/internal/provider"
	"github.com/Jayleonc/ai-gateway/pkg/openai"
)

// ModelsHandler 模型列表处理器
type ModelsHandler struct {
	providerRegistry provider.Registry
}

// NewModelsHandler 创建模型处理器
func NewModelsHandler(pr provider.Registry) *ModelsHandler {
	return &ModelsHandler{
		providerRegistry: pr,
	}
}

// List 列出所有可用模型
func (h *ModelsHandler) List(c *gin.Context) {
	providers := h.providerRegistry.List()

	var models []Model
	for _, p := range providers {
		for _, m := range p.SupportedModels() {
			models = append(models, Model{
				ID:      m,
				Object:  openai.ObjectModel,
				Created: time.Now().Unix(),
				OwnedBy: p.Name(),
			})
		}
	}

	c.JSON(http.StatusOK, ModelList{
		Object: openai.ObjectList,
		Data:   models,
	})
}

// Get 获取单个模型信息
func (h *ModelsHandler) Get(c *gin.Context) {
	modelID := c.Param("model")

	p, ok := h.providerRegistry.GetByModel(modelID)
	if !ok {
		WriteOpenAIError(c, coreerrors.NewAPIErrorWithStatus("invalid_request", "model not found", "invalid_request_error", http.StatusNotFound))
		return
	}

	c.JSON(http.StatusOK, Model{
		ID:      modelID,
		Object:  openai.ObjectModel,
		Created: time.Now().Unix(),
		OwnedBy: p.Name(),
	})
}

// Model 模型信息
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ModelList 模型列表
type ModelList struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}
