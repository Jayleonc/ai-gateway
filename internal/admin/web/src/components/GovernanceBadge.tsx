import { Tag } from 'antd';

interface GovernanceBadgeProps {
  status: string;
}

const statusConfig: Record<string, { color: string; label: string }> = {
  active: { color: 'green', label: '正常' },
  limited: { color: 'orange', label: '受限' },
  exhausted: { color: 'red', label: '耗尽' },
};

export function GovernanceBadge({ status }: GovernanceBadgeProps) {
  const config = statusConfig[status] || { color: 'default', label: status };
  return <Tag color={config.color}>{config.label}</Tag>;
}
