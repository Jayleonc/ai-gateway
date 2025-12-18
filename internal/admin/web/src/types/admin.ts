export interface AllowanceView {
  allowed_models: string[];
  quota_limit: number;
  rate_limit: number;
}

export interface QuotaView {
  used: number;
  remaining: number;
  total: number;
}

export interface UsageSummaryView {
  recent_requests: number;
  total_confirmed_tokens: number;
  has_quota_exceeded: boolean;
  has_partial: boolean;
}

export interface GovernanceView {
  status: string;
  reason: string;
}

export interface KeyOverview {
  api_key_id: string;
  allowance: AllowanceView;
  quota: QuotaView;
  usage_summary: UsageSummaryView;
  governance: GovernanceView;
}

export interface UsageView {
  request_id: string;
  model: string;
  status: string;
  end_reason: string;
  confirmed_tokens: number;
  started_at: number;
  ended_at: number;
}

export interface UsageListResponse {
  api_key_id: string;
  usages: UsageView[];
  total: number;
}

export interface GovernanceState {
  api_key_id: string;
  status: string;
  reason: string;
}

export interface GovernanceSummary {
  total_keys: number;
  active_keys: number;
  limited_keys: number;
  exhausted_keys: number;
}

export interface ResetQuotaResponse {
  api_key_id: string;
  success: boolean;
  new_remaining: number;
  message: string;
}
