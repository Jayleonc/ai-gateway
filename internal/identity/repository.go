package identity

import (
	"context"
	"sync"
	"time"

	coreerrors "github.com/Jayleonc/ai-gateway/internal/core/errors"
)

// APIKeyInfo API Key 详细信息
type APIKeyInfo struct {
	ID            string
	KeyHash       string
	ProjectID     string
	TenantID      string
	CostCenter    string
	AllowedModels []string
	RateLimit     int
	QuotaLimit    int64
	Labels        map[string]string
	Status        string
	ExpiresAt     *time.Time
	CreatedAt     time.Time
}

// APIKeyRepository API Key 存储接口
type APIKeyRepository interface {
	GetByKey(ctx context.Context, key string) (*APIKeyInfo, error)
	GetByID(ctx context.Context, id string) (*APIKeyInfo, error)
}

// apiKeyRepository API Key 存储实现（stub）
type apiKeyRepository struct {
	mu    sync.RWMutex
	byKey map[string]*APIKeyInfo
	byID  map[string]*APIKeyInfo
}

// NewAPIKeyRepository 创建 API Key 存储
func NewAPIKeyRepository() APIKeyRepository {
	r := &apiKeyRepository{
		byKey: make(map[string]*APIKeyInfo),
		byID:  make(map[string]*APIKeyInfo),
	}

	now := time.Now()
	sample := []*APIKeyInfo{
		{
			ID:            "gw-key-001",
			ProjectID:     "project-001",
			TenantID:      "tenant-001",
			CostCenter:    "default",
			AllowedModels: []string{"gpt-4", "gpt-3.5-turbo"},
			RateLimit:     100,
			QuotaLimit:    1000000,
			Labels:        map[string]string{},
			Status:        "active",
			CreatedAt:     now,
		},
		{
			ID:            "gw-key-002",
			ProjectID:     "project-002",
			TenantID:      "tenant-002",
			CostCenter:    "default",
			AllowedModels: []string{"gpt-4", "gpt-3.5-turbo"},
			RateLimit:     50,
			QuotaLimit:    500000,
			Labels:        map[string]string{},
			Status:        "active",
			CreatedAt:     now,
		},
	}

	seedKeys := []string{"sk-gw-001", "sk-gw-002"}
	for i, key := range seedKeys {
		info := sample[i]
		r.byKey[key] = info
		r.byID[info.ID] = info
	}

	return r
}

func (r *apiKeyRepository) GetByKey(ctx context.Context, key string) (*APIKeyInfo, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	info, ok := r.byKey[key]
	if !ok {
		return nil, coreerrors.ErrNotFound
	}
	return info, nil
}

func (r *apiKeyRepository) GetByID(ctx context.Context, id string) (*APIKeyInfo, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	info, ok := r.byID[id]
	if !ok {
		return nil, coreerrors.ErrNotFound
	}
	return info, nil
}

var APIKeyMap = []string{
	"sk-gw-001",
	"sk-gw-002",
}
