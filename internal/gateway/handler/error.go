package handler

import (
	"github.com/gin-gonic/gin"

	coreerrors "github.com/Jayleonc/ai-gateway/internal/core/errors"
	"github.com/Jayleonc/ai-gateway/pkg/openai"
)

func WriteOpenAIError(c *gin.Context, err error) {
	apiErr := coreerrors.AsAPIError(err)
	if apiErr == nil {
		c.JSON(500, openai.ErrorResponse{
			Error: &openai.ErrorDetail{
				Message: "internal server error",
				Type:    "server_error",
			},
		})
		return
	}

	code := apiErr.Code()
	c.JSON(apiErr.HTTPStatus(), openai.ErrorResponse{
		Error: &openai.ErrorDetail{
			Message: apiErr.Message(),
			Type:    apiErr.Type(),
			Code:    &code,
		},
	})
}
