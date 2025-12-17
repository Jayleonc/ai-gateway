package metering

import (
	"sort"
)

type UsageQuery interface {
	ListByAPIKey(apiKeyID string, limit int) ([]*UsageRecord, error)
	GetByRequestID(requestID string) (*UsageRecord, error)
}

type InMemoryUsageQuery struct {
	rec *InMemoryRecorder
}

func NewInMemoryUsageQuery(rec *InMemoryRecorder) *InMemoryUsageQuery {
	return &InMemoryUsageQuery{rec: rec}
}

func (q *InMemoryUsageQuery) ListByAPIKey(apiKeyID string, limit int) ([]*UsageRecord, error) {
	if q == nil || q.rec == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}

	all := q.rec.All()
	if len(all) == 0 {
		return nil, nil
	}

	out := make([]*UsageRecord, 0, len(all))
	for _, r := range all {
		if r == nil {
			continue
		}
		if apiKeyID != "" && r.APIKeyID != apiKeyID {
			continue
		}
		out = append(out, r)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].RequestedAt.After(out[j].RequestedAt)
	})

	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (q *InMemoryUsageQuery) GetByRequestID(requestID string) (*UsageRecord, error) {
	if q == nil || q.rec == nil {
		return nil, nil
	}
	return q.rec.GetByRequestID(requestID), nil
}

var _ UsageQuery = (*InMemoryUsageQuery)(nil)
