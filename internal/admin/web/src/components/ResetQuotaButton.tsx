import { Button, message, Popconfirm } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useState } from 'react';
import { resetQuota } from '../api/admin';

interface ResetQuotaButtonProps {
  apiKeyId: string;
  onSuccess?: () => void;
  onReset?: () => Promise<void>;
  loading?: boolean;
}

export function ResetQuotaButton({ apiKeyId, onSuccess, onReset, loading }: ResetQuotaButtonProps) {
  const [innerLoading, setInnerLoading] = useState(false);
  const isLoading = loading ?? innerLoading;

  const handleReset = async () => {
    setInnerLoading(true);
    try {
      if (onReset) {
        await onReset();
        return;
      }

      const result = await resetQuota(apiKeyId);
      if (!result.success) {
        message.error('重置失败');
        return;
      }

      message.success(result.message);
      onSuccess?.();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '重置失败');
    } finally {
      setInnerLoading(false);
    }
  };

  return (
    <Popconfirm
      title="重置配额"
      description="确认要重置该 API Key 的配额吗？"
      onConfirm={handleReset}
      okText="确认"
      cancelText="取消"
    >
      <Button
        type="primary"
        icon={<ReloadOutlined />}
        loading={isLoading}
        danger
      >
        重置配额
      </Button>
    </Popconfirm>
  );
}
