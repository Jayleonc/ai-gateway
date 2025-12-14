package openai

import (
	"net/http"

	coreerrors "github.com/Jayleonc/ai-gateway/internal/core/errors"
	"github.com/Jayleonc/ai-gateway/internal/gateway/streaming"
	openaitypes "github.com/Jayleonc/ai-gateway/pkg/openai"
)

func MapError(err error) ErrorPlan {
	apiErr := coreerrors.AsAPIError(err)
	if apiErr == nil {
		return ErrorPlan{
			HTTPStatus: http.StatusInternalServerError,
			Body: openaitypes.ErrorResponse{
				Error: &openaitypes.ErrorDetail{
					Message: coreerrors.ErrInternalServer.Error(),
					Type:    coreerrors.OpenAIErrorTypeServer,
				},
			},
		}
	}

	code := apiErr.Code()
	return ErrorPlan{
		HTTPStatus: apiErr.HTTPStatus(),
		Body: openaitypes.ErrorResponse{
			Error: &openaitypes.ErrorDetail{
				Message: apiErr.Message(),
				Type:    apiErr.Type(),
				Code:    &code,
			},
		},
	}
}

func MapStreamEnd(ctx *streaming.StreamingContext, runErr error) StreamEndPlan {
	if ctx == nil {
		p := MapError(coreerrors.ErrInternalServer)
		return StreamEndPlan{Action: StreamEndActionWriteJSONError, Error: &p}
	}

	started := ctx.IsStarted()
	endedNormally := ctx.EndReason == streaming.EndReasonStop || ctx.EndReason == streaming.EndReasonLength
	if started {
		if endedNormally && runErr == nil {
			return StreamEndPlan{Action: StreamEndActionNone}
		}
		return StreamEndPlan{Action: StreamEndActionCloseStream, HTTPStatus: http.StatusOK}
	}

	mappedErr := mapStreamingEndToError(ctx, runErr)
	p := MapError(mappedErr)
	return StreamEndPlan{Action: StreamEndActionWriteJSONError, Error: &p}
}

func mapStreamingEndToError(ctx *streaming.StreamingContext, runErr error) error {
	if ctx == nil {
		return runErr
	}

	switch ctx.EndReason {
	case streaming.EndReasonQuotaExceeded:
		return coreerrors.ErrQuotaExceeded
	case streaming.EndReasonError:
		return coreerrors.ErrProviderError
	case streaming.EndReasonInternalError:
		return coreerrors.ErrInternalServer
	case streaming.EndReasonClientDisconnect:
		return coreerrors.ErrClientDisconnect
	default:
		if runErr != nil {
			return runErr
		}
		return coreerrors.ErrInternalServer
	}
}
