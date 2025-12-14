package handler

import (
	"github.com/gin-gonic/gin"

	transportopenai "github.com/Jayleonc/ai-gateway/internal/transport/openai"
)

func WriteOpenAIError(c *gin.Context, err error) {
	plan := transportopenai.MapError(err)
	c.JSON(plan.HTTPStatus, plan.Body)
}
