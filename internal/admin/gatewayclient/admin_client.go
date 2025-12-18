package gatewayclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type GatewayAdminClient struct {
	gatewayURL string
	httpClient *http.Client
}

func NewGatewayAdminClient(gatewayURL string) *GatewayAdminClient {
	return &GatewayAdminClient{
		gatewayURL: gatewayURL,
		httpClient: &http.Client{},
	}
}

type QuotaFact struct {
	APIKeyID  string `json:"api_key_id"`
	Used      int64  `json:"used"`
	Remaining int64  `json:"remaining"`
	Total     int64  `json:"total"`
}

type UsageFact struct {
	RequestID       string `json:"request_id"`
	Model           string `json:"model"`
	ConfirmedTokens int64  `json:"confirmed_tokens"`
	Status          string `json:"status"`
	EndReason       string `json:"end_reason"`
}

type UsagesResponse struct {
	APIKeyID string      `json:"api_key_id"`
	Usages   []UsageFact `json:"usages"`
	Total    int         `json:"total"`
}

// GetQuota retrieves the quota fact for a given API key from Gateway
func (c *GatewayAdminClient) GetQuota(ctx context.Context, apiKeyID string) (*QuotaFact, error) {
	if apiKeyID == "" {
		return nil, fmt.Errorf("api_key_id is required")
	}

	url := fmt.Sprintf("%s/internal/admin/keys/%s/quota", c.gatewayURL, apiKeyID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Gateway: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Gateway returned status %d: %s", resp.StatusCode, string(body))
	}

	var quota QuotaFact
	if err := json.NewDecoder(resp.Body).Decode(&quota); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &quota, nil
}

// GetUsages retrieves usage records for a given API key from Gateway
func (c *GatewayAdminClient) GetUsages(ctx context.Context, apiKeyID string, limit int) ([]UsageFact, error) {
	if apiKeyID == "" {
		return nil, fmt.Errorf("api_key_id is required")
	}

	url := fmt.Sprintf("%s/internal/admin/keys/%s/usages", c.gatewayURL, apiKeyID)
	if limit > 0 {
		url = fmt.Sprintf("%s?limit=%d", url, limit)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Gateway: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Gateway returned status %d: %s", resp.StatusCode, string(body))
	}

	var usagesResp UsagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&usagesResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return usagesResp.Usages, nil
}
