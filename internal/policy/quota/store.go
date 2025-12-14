package quota

import "sync"

type QuotaStore interface {
	Remaining(apiKeyID string) int64
	SetRemaining(apiKeyID string, remaining int64)
	TryConsume(apiKeyID string, tokens int64) bool
}

type inMemoryStore struct {
	mu      sync.RWMutex
	records map[string]int64
}

func NewInMemoryStore() QuotaStore {
	return &inMemoryStore{
		records: make(map[string]int64),
	}
}

func (i *inMemoryStore) Remaining(apiKeyID string) int64 {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.records[apiKeyID]
}

func (i *inMemoryStore) SetRemaining(apiKeyID string, remaining int64) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.records == nil {
		i.records = make(map[string]int64)
	}
	i.records[apiKeyID] = remaining
}

func (i *inMemoryStore) TryConsume(apiKeyID string, tokens int64) bool {
	if tokens <= 0 {
		return true
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	if i.records == nil {
		i.records = make(map[string]int64)
	}

	rem := i.records[apiKeyID]
	if rem < tokens {
		return false
	}
	i.records[apiKeyID] = rem - tokens
	return true
}
