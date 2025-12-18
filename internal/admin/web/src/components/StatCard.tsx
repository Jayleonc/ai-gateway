import { Card, Statistic, Typography } from 'antd';
import type { ReactNode } from 'react';

interface StatCardProps {
  title: string;
  value: number | string;
  icon?: ReactNode;
  color?: string;
  description?: string;
  active?: boolean;
}

export function StatCard({ title, value, icon, color, description, active }: StatCardProps) {
  return (
    <Card>
      <div
        style={{
          opacity: active ? 1 : 0,
          transform: active ? 'translateY(0px)' : 'translateY(8px)',
          transition: 'opacity 220ms ease, transform 220ms ease',
        }}
      >
        <Statistic
          title={title}
          value={value}
          prefix={icon}
          valueStyle={{ color }}
        />
        {description ? (
          <Typography.Text type="secondary">{description}</Typography.Text>
        ) : null}
      </div>
    </Card>
  );
}
