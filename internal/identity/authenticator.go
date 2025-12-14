package identity

import (
	"context"
	"strings"
	"time"

	"github.com/Jayleonc/ai-gateway/internal/core/errors"
)

// Authenticator API Key 认证器接口
type Authenticator interface {
	Authenticate(ctx context.Context, apiKey string) (*RequestContext, error)
}

// authenticator 认证器实现
type authenticator struct {
	repo     APIKeyRepository
	resolver Resolver
}

// NewAuthenticator 创建认证器
func NewAuthenticator(repo APIKeyRepository) Authenticator {
	return &authenticator{repo: repo, resolver: NewResolver()}
}

// Authenticate 校验 API Key
func (a *authenticator) Authenticate(ctx context.Context, apiKey string) (*RequestContext, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, errors.ErrInvalidAPIKey
	}

	keyInfo, err := a.repo.GetByKey(ctx, apiKey)
	if err != nil {
		return nil, errors.ErrInvalidAPIKey
	}
	if keyInfo == nil {
		return nil, errors.ErrInvalidAPIKey
	}

	if keyInfo.ExpiresAt != nil && time.Now().After(*keyInfo.ExpiresAt) {
		return nil, errors.ErrAPIKeyExpired
	}
	if keyInfo.Status != "" && keyInfo.Status != "active" {
		return nil, errors.ErrAPIKeyDisabled
	}

	reqCtx, err := a.resolver.Resolve(ctx, keyInfo)
	if err != nil {
		return nil, err
	}
	if reqCtx == nil {
		return nil, errors.ErrInvalidAPIKey
	}
	if reqCtx.Labels == nil {
		reqCtx.Labels = make(map[string]string)
	}

	return reqCtx, nil
}
