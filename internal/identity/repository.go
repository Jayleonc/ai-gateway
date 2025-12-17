package identity

import (
	"context"
	"sync"
	"time"

	coreerrors "github.com/Jayleonc/ai-gateway/internal/core/errors"
)

// APIKeyInfo API Key 详细信息
//
// APIKeyInfo 表示 identity 级别的 Allowance（使用许可）配置。
// 这些字段定义的是“使用许可（Allowance）”，而不是“实时 enforcement 来源”或“计费（Billing）”。
type APIKeyInfo struct {
	ID         string
	KeyHash    string
	ProjectID  string
	TenantID   string
	CostCenter string

	// AllowedModels 定义该 key 被允许访问哪些模型（能力许可）。
	AllowedModels []string
	// QuotaLimit 定义该 key 被授予的最大使用预算（预算许可）。
	// NOTE：QuotaLimit 是 Allowance 配置，而不是 enforcement source。
	QuotaLimit int64
	// RateLimit 定义该 key 被授予的使用速率许可（时序/速率许可）。
	// NOTE：Phase 6 不要求也不实现 RateLimit 的强制执行。
	RateLimit int

	Labels    map[string]string
	Status    string
	ExpiresAt *time.Time
	CreatedAt time.Time
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
