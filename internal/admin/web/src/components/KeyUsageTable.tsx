import { Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type { UsageView } from '../types/admin';

interface KeyUsageTableProps {
  usages: UsageView[];
  loading?: boolean;
}

const statusColors: Record<string, string> = {
  completed: 'green',
  streaming: 'blue',
  failed: 'red',
  partial: 'orange',
};

const statusLabels: Record<string, string> = {
  completed: '已完成',
  streaming: '流式中',
  failed: '失败',
  partial: '部分完成',
};

const endReasonLabels: Record<string, string> = {
  stop: '正常结束',
  quota_exceeded: '配额耗尽',
  length: '长度限制',
  content_filter: '内容过滤',
  error: '错误',
  partial: '部分完成',
};

const columns: ColumnsType<UsageView> = [
  {
    title: '请求 ID',
    dataIndex: 'request_id',
    key: 'request_id',
    ellipsis: true,
    width: 200,
  },
  {
    title: '模型',
    dataIndex: 'model',
    key: 'model',
    width: 150,
  },
  {
    title: 'Token',
    dataIndex: 'confirmed_tokens',
    key: 'confirmed_tokens',
    width: 100,
    render: (tokens: number) => tokens.toLocaleString(),
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    width: 100,
    render: (status: string) => (
      <Tag color={statusColors[status] || 'default'}>
        {statusLabels[status] || status}
      </Tag>
    ),
  },
  {
    title: '结束原因',
    dataIndex: 'end_reason',
    key: 'end_reason',
    width: 120,
    render: (reason: string) => endReasonLabels[reason] || reason || '-',
  },
];

export function KeyUsageTable({ usages, loading }: KeyUsageTableProps) {
  return (
    <Table
      columns={columns}
      dataSource={usages}
      rowKey="request_id"
      loading={loading}
      size="small"
      pagination={{ pageSize: 10 }}
    />
  );
}
