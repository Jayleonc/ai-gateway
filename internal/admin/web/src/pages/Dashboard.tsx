import { useEffect, useMemo, useState } from 'react';
import { Card, Col, Row, Spin, Alert, Typography, Input, Button, Space } from 'antd';
import {
  KeyOutlined,
  CheckCircleOutlined,
  WarningOutlined,
  StopOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { StatCard } from '../components/StatCard';
import { getGovernanceSummary } from '../api/admin';
import type { GovernanceSummary } from '../types/admin';

const { Title } = Typography;

export function Dashboard() {
  const navigate = useNavigate();
  const [summary, setSummary] = useState<GovernanceSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [animate, setAnimate] = useState(false);
  const [keyId, setKeyId] = useState('gw-key-001');

  useEffect(() => {
    loadSummary();
  }, []);

  useEffect(() => {
    if (!loading && summary) {
      const t = window.setTimeout(() => setAnimate(true), 60);
      return () => window.clearTimeout(t);
    }
    return;
  }, [loading, summary]);

  const jumpPath = useMemo(() => {
    const trimmed = keyId.trim();
    if (!trimmed) return '';
    return `/admin/keys/${trimmed}`;
  }, [keyId]);

  const loadSummary = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getGovernanceSummary();
      setSummary(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载治理总览失败');
    } finally {
      setLoading(false);
    }
  };

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

  if (!summary) {
    return null;
  }

  return (
    <div style={{ padding: 24 }}>
      <Title level={2}>平台治理总览</Title>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="Key 总数"
            value={summary.total_keys}
            icon={<KeyOutlined />}
            color="#1890ff"
            description="平台已知的全部 API Key"
            active={animate}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="正常 Key"
            value={summary.active_keys}
            icon={<CheckCircleOutlined />}
            color="#52c41a"
            description="配额充足，可正常请求"
            active={animate}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="受限 Key"
            value={summary.limited_keys}
            icon={<WarningOutlined />}
            color="#faad14"
            description="可能出现部分完成或其他约束"
            active={animate}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="耗尽 Key"
            value={summary.exhausted_keys}
            icon={<StopOutlined />}
            color="#ff4d4f"
            description="剩余配额为 0"
            active={animate}
          />
        </Col>
      </Row>

      <Card style={{ marginTop: 24 }}>
        <Title level={4}>跳转到 Key</Title>
        <Space direction="vertical" style={{ width: '100%' }} size={12}>
          <Typography.Text type="secondary">
            输入 API Key ID 以打开对应的治理视图。
          </Typography.Text>
          <Space.Compact style={{ width: '100%' }}>
            <Input
              placeholder="API Key ID（例如：gw-key-001）"
              value={keyId}
              onChange={(e) => setKeyId(e.target.value)}
              onPressEnter={() => {
                if (jumpPath) navigate(jumpPath);
              }}
            />
            <Button
              type="primary"
              disabled={!jumpPath}
              onClick={() => {
                if (jumpPath) navigate(jumpPath);
              }}
            >
              打开
            </Button>
          </Space.Compact>
          <Typography.Text type="secondary">
            URL 格式：<code>/admin/keys/:api_key_id</code>
          </Typography.Text>
        </Space>
      </Card>
    </div>
  );
}
