import type {
  GovernanceSummary,
  KeyOverview,
  UsageListResponse,
  ResetQuotaResponse,
} from '../types/admin';

const BASE_URL = '/admin';

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: '请求失败' }));
    throw new Error(error.error || `HTTP ${response.status}`);
  }

  return response.json();
}

export async function getGovernanceSummary(): Promise<GovernanceSummary> {
  return request<GovernanceSummary>(`${BASE_URL}/governance/summary`);
}

export async function getKeyOverview(apiKeyId: string): Promise<KeyOverview> {
  return request<KeyOverview>(`${BASE_URL}/keys/${apiKeyId}/overview`);
}

export async function getKeyUsages(apiKeyId: string, limit = 20): Promise<UsageListResponse> {
  return request<UsageListResponse>(`${BASE_URL}/keys/${apiKeyId}/usages?limit=${limit}`);
}

export async function resetQuota(apiKeyId: string): Promise<ResetQuotaResponse> {
  return request<ResetQuotaResponse>(`${BASE_URL}/keys/${apiKeyId}/reset-quota`, {
    method: 'POST',
  });
}
