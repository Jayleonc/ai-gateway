import { useEffect, useMemo, useRef, useState, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import {
  Card,
  Col,
  Row,
  Spin,
  Alert,
  Typography,
  Progress,
  Descriptions,
  Tag,
  Space,
  Divider,
  message,
} from 'antd';
import { GovernanceBadge } from '../components/GovernanceBadge';
import { KeyUsageTable } from '../components/KeyUsageTable';
import { ResetQuotaButton } from '../components/ResetQuotaButton';
import { StatCard } from '../components/StatCard';
import { getKeyOverview, getKeyUsages, resetQuota } from '../api/admin';
import type { KeyOverview, UsageView } from '../types/admin';

const { Title } = Typography;

export function KeyDetail() {
  const { api_key_id } = useParams<{ api_key_id: string }>();
  const [overview, setOverview] = useState<KeyOverview | null>(null);
  const [usages, setUsages] = useState<UsageView[]>([]);
  const [loading, setLoading] = useState(true);
  const [usagesLoading, setUsagesLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [resetting, setResetting] = useState(false);
  const [highlightQuota, setHighlightQuota] = useState(false);
  const [highlightGovernance, setHighlightGovernance] = useState(false);
  const usageLimit = 50;

  const governanceStatusLabel = useCallback((status: string) => {
    switch (status) {
      case 'active':
        return '正常';
      case 'limited':
        return '受限';
      case 'exhausted':
        return '耗尽';
      default:
        return status;
    }
  }, []);

  const lastSnapshotRef = useRef<{
    remaining: number;
    governanceStatus: string;
  } | null>(null);
  const [lastResetNote, setLastResetNote] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    if (!api_key_id) return;

    setLoading(true);
    setError(null);
    try {
      const [overviewData, usagesData] = await Promise.all([
        getKeyOverview(api_key_id),
        getKeyUsages(api_key_id, usageLimit),
      ]);
      setOverview(overviewData);
      setUsages(usagesData.usages);
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载数据失败');
    } finally {
      setLoading(false);
    }
  }, [api_key_id]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const usageImpact = useMemo(() => {
    const totalRequests = usages.length;
    let totalConfirmedTokens = 0;
    let partialRequests = 0;
    let quotaExceededRequests = 0;

    for (const u of usages) {
      totalConfirmedTokens += u.confirmed_tokens;
      if (u.status === 'partial') partialRequests += 1;
      if (u.end_reason === 'quota_exceeded') quotaExceededRequests += 1;
    }

    const partialRatio = totalRequests > 0 ? partialRequests / totalRequests : 0;
    const quotaExceededRatio = totalRequests > 0 ? quotaExceededRequests / totalRequests : 0;

    let trendText = '暂无最近使用记录。';
    if (totalRequests > 0) {
      if (quotaExceededRequests > 0) {
        trendText = `检测到配额拦截（最近请求中 ${(quotaExceededRatio * 100).toFixed(0)}% 因配额结束）。`;
      } else if (partialRatio >= 0.3) {
        trendText = `检测到较高的部分完成比例（${(partialRatio * 100).toFixed(0)}%）。`;
      } else {
        trendText = '最近使用情况看起来正常。';
      }
    }

    return {
      totalRequests,
      totalConfirmedTokens,
      partialRequests,
      quotaExceededRequests,
      trendText,
    };
  }, [usages]);

  const MODEL_COST_PER_1K_TOKENS: Record<string, number> = useMemo(
    () => ({
      'gpt-4': 0.03,
      'gpt-3.5-turbo': 0.002,
    }),
    []
  );

  const estimatedCost = useMemo(() => {
    const byModelTokens = new Map<string, number>();
    let totalTokens = 0;
    for (const u of usages) {
      totalTokens += u.confirmed_tokens;
      byModelTokens.set(u.model, (byModelTokens.get(u.model) ?? 0) + u.confirmed_tokens);
    }

    let knownTokens = 0;
    let totalCost = 0;
    for (const [model, tokens] of byModelTokens.entries()) {
      const rate = MODEL_COST_PER_1K_TOKENS[model];
      if (typeof rate === 'number') {
        knownTokens += tokens;
        totalCost += (tokens / 1000) * rate;
      }
    }

    const coverage = totalTokens > 0 ? knownTokens / totalTokens : 0;
    return {
      totalTokens,
      knownTokens,
      coverage,
      totalCost,
    };
  }, [MODEL_COST_PER_1K_TOKENS, usages]);

  const governanceExplanation = useMemo(() => {
    if (!overview) return [] as string[];

    const lines: string[] = [];
    const remaining = overview.quota.remaining;
    const quotaLimit = overview.allowance.quota_limit;
    const status = overview.governance.status;

    if (status === 'active') {
      lines.push('当前 Key 状态为“正常”，因为剩余配额充足，可以继续发起请求。');
    } else if (status === 'limited') {
      lines.push('当前 Key 状态为“受限”，平台检测到治理约束，可能影响请求完成度。');
    } else if (status === 'exhausted') {
      lines.push('当前 Key 状态为“耗尽”，因为剩余配额已用完。');
    } else {
      lines.push(`当前 Key 处于治理状态：${status}。`);
    }

    if (overview.governance.reason) {
      lines.push(`治理原因：${overview.governance.reason}`);
    }

    if (remaining <= 0) {
      lines.push('剩余配额为 0，新请求可能会因为配额拦截而提前结束。');
    } else if (remaining < Math.max(1, Math.floor(quotaLimit * 0.1))) {
      lines.push('剩余配额较低，更容易出现部分完成。');
    }

    if (overview.usage_summary.has_partial) {
      lines.push('最近检测到部分完成的请求，可能与配额拦截有关。');
    }
    if (overview.usage_summary.has_quota_exceeded) {
      lines.push('最近使用记录中检测到配额耗尽信号。');
    }

    lines.push('“重置配额”会将剩余配额恢复到配置的配额上限。');
    return lines;
  }, [overview]);

  const handleReset = useCallback(async () => {
    if (!api_key_id || !overview) return;

    lastSnapshotRef.current = {
      remaining: overview.quota.remaining,
      governanceStatus: overview.governance.status,
    };

    setResetting(true);
    setLastResetNote(null);
    try {
      const result = await resetQuota(api_key_id);
      if (!result.success) {
        message.error('重置失败');
        return;
      }

      message.success('已提交重置配额请求');

      setUsagesLoading(true);
      const [overviewData, usagesData] = await Promise.all([
        getKeyOverview(api_key_id),
        getKeyUsages(api_key_id, usageLimit),
      ]);

      setOverview(overviewData);
      setUsages(usagesData.usages);

      const before = lastSnapshotRef.current;
      if (before) {
        const quotaChanged = before.remaining !== overviewData.quota.remaining;
        const governanceChanged = before.governanceStatus !== overviewData.governance.status;
        setHighlightQuota(true);
        setHighlightGovernance(true);

        const noteParts: string[] = [];
        noteParts.push(
          `剩余配额从 ${before.remaining.toLocaleString()} 变更为 ${overviewData.quota.remaining.toLocaleString()}。`
        );
        if (governanceChanged) {
          noteParts.push(
            `治理状态从 ${governanceStatusLabel(before.governanceStatus)} 变更为 ${governanceStatusLabel(overviewData.governance.status)}。`
          );
        } else {
          noteParts.push(`治理状态保持为 ${governanceStatusLabel(overviewData.governance.status)}。`);
        }

        if (!quotaChanged) {
          noteParts.push('重置后剩余配额未发生变化（可能已在上限或本次无变化）。');
        }
        setLastResetNote(noteParts.join(' '));
      } else {
        setHighlightQuota(true);
        setHighlightGovernance(true);
        setLastResetNote('已触发配额重置并刷新数据。');
      }
    } catch (err) {
      message.error(err instanceof Error ? err.message : '重置失败');
    } finally {
      setResetting(false);
      setUsagesLoading(false);

      window.setTimeout(() => setHighlightQuota(false), 4000);
      window.setTimeout(() => setHighlightGovernance(false), 4000);
    }
  }, [api_key_id, overview, usageLimit]);

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: 100 }}>
        <Spin size="large" />
      </div>
    );
  }

  if (error) {
    return (
      <Alert
        message="错误"
        description={error}
        type="error"
        showIcon
        style={{ margin: 24 }}
      />
    );
  }

  if (!overview) {
    return null;
  }

  const quotaPercent =
    overview.quota.total > 0
      ? Math.round((overview.quota.used / overview.quota.total) * 100)
      : 0;

  const quotaStatus =
    quotaPercent >= 100 ? 'exception' : quotaPercent >= 80 ? 'normal' : 'success';

  return (
    <div style={{ padding: 24 }}>
      <Title level={2}>Key 详情：{api_key_id}</Title>

      {lastResetNote ? (
        <Alert
          type="success"
          showIcon
          message="治理动作已执行"
          description={lastResetNote}
          style={{ marginBottom: 16 }}
        />
      ) : null}

      <Row gutter={[16, 16]}>
        {/* A. Key Basic Info */}
        <Col xs={24} lg={12}>
          <Card title="Key 信息" size="small">
            <Descriptions column={1} size="small">
              <Descriptions.Item label="API Key ID">
                <code>{overview.api_key_id}</code>
              </Descriptions.Item>
              <Descriptions.Item label="允许模型">
                <Space wrap>
                  {overview.allowance.allowed_models.map((model: string) => (
                    <Tag key={model}>{model}</Tag>
                  ))}
                </Space>
              </Descriptions.Item>
              <Descriptions.Item label="速率限制">
                {overview.allowance.rate_limit} 次/分钟
              </Descriptions.Item>
              <Descriptions.Item label="配额上限">
                {overview.allowance.quota_limit.toLocaleString()} Token
              </Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>

        {/* B. Quota Status */}
        <Col xs={24} lg={12}>
          <Card
            title="配额状态"
            size="small"
            style={
              highlightQuota
                ? {
                    borderColor: '#52c41a',
                    boxShadow: '0 0 0 2px rgba(82, 196, 26, 0.15)',
                  }
                : undefined
            }
          >
            <Progress
              percent={quotaPercent}
              status={quotaStatus}
              format={() => `${quotaPercent}%`}
            />
            <Descriptions column={1} size="small" style={{ marginTop: 16 }}>
              <Descriptions.Item label="已使用">
                {overview.quota.used.toLocaleString()} Token
              </Descriptions.Item>
              <Descriptions.Item label="剩余">
                {overview.quota.remaining.toLocaleString()} Token
              </Descriptions.Item>
              <Descriptions.Item label="总量">
                {overview.quota.total.toLocaleString()} Token
              </Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>

        {/* C. Governance Status */}
        <Col xs={24} lg={12}>
          <Card
            title="治理状态"
            size="small"
            style={
              highlightGovernance
                ? {
                    borderColor: '#1890ff',
                    boxShadow: '0 0 0 2px rgba(24, 144, 255, 0.15)',
                  }
                : undefined
            }
          >
            <Descriptions column={1} size="small">
              <Descriptions.Item label="状态">
                <GovernanceBadge status={overview.governance.status} />
              </Descriptions.Item>
              <Descriptions.Item label="原因">
                {overview.governance.reason || '-'}
              </Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>

        {/* 3.1 Usage Impact Visualization */}
        <Col xs={24}>
          <Card title={`使用影响（最近 ${Math.min(usageLimit, usageImpact.totalRequests)} 次请求）`} size="small">
            <Row gutter={[16, 16]}>
              <Col xs={24} sm={12} lg={6}>
                <StatCard
                  title="确认 Token 总量"
                  value={usageImpact.totalConfirmedTokens.toLocaleString()}
                  description="最近记录中确认 Token 的总和"
                  active
                />
              </Col>
              <Col xs={24} sm={12} lg={6}>
                <StatCard
                  title="请求数"
                  value={usageImpact.totalRequests}
                  description="最近使用记录的条数"
                  active
                />
              </Col>
              <Col xs={24} sm={12} lg={6}>
                <StatCard
                  title="部分完成"
                  value={usageImpact.partialRequests}
                  description="未完整完成即结束的请求"
                  active
                />
              </Col>
              <Col xs={24} sm={12} lg={6}>
                <StatCard
                  title="配额耗尽"
                  value={usageImpact.quotaExceededRequests}
                  description="因配额拦截而结束的请求"
                  active
                />
              </Col>
            </Row>

            <Divider style={{ margin: '12px 0' }} />
            <Typography.Text type="secondary">{usageImpact.trendText}</Typography.Text>
          </Card>
        </Col>

        {/* 3.2 Cost Estimation (display-only) */}
        <Col xs={24}>
          <Card title="使用成本估算（仅展示）" size="small">
            <Row gutter={[16, 16]}>
              <Col xs={24} sm={12} lg={8}>
                <StatCard
                  title="预估成本"
                  value={`$${estimatedCost.totalCost.toFixed(4)}`}
                  description="最近使用的前端估算成本（仅展示）"
                  active
                />
              </Col>
              <Col xs={24} sm={12} lg={8}>
                <StatCard
                  title="已知模型覆盖率"
                  value={`${(estimatedCost.coverage * 100).toFixed(0)}%`}
                  description="可映射到已知模型费率的 Token 占比"
                  active
                />
              </Col>
              <Col xs={24} sm={12} lg={8}>
                <StatCard
                  title="纳入估算的 Token"
                  value={estimatedCost.knownTokens.toLocaleString()}
                  description="参与估算的 Token 数量"
                  active
                />
              </Col>
            </Row>
            <Divider style={{ margin: '12px 0' }} />
            <Typography.Text type="secondary">
              该估算仅用于展示，不代表计费。
            </Typography.Text>
          </Card>
        </Col>

        {/* 3.4 Governance Explanation Panel */}
        <Col xs={24}>
          <Card title="治理解释" size="small">
            <Space direction="vertical" size={8} style={{ width: '100%' }}>
              {governanceExplanation.map((line) => (
                <Typography.Text key={line}>{line}</Typography.Text>
              ))}
            </Space>
          </Card>
        </Col>

        {/* E. Governance Actions */}
        <Col xs={24} lg={12}>
          <Card title="治理动作" size="small">
            <ResetQuotaButton
              apiKeyId={overview.api_key_id}
              onReset={handleReset}
              loading={resetting}
            />
          </Card>
        </Col>

        {/* D. Recent Usage */}
        <Col xs={24}>
          <Card title="最近使用记录" size="small">
            <KeyUsageTable usages={usages} loading={usagesLoading} />
          </Card>
        </Col>
      </Row>
    </div>
  );
}
