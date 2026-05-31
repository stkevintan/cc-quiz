import { useEffect, useState } from 'react';
import { Alert, Button, Card, Descriptions, Space, Spin, Tag, Typography } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { getHealth } from '../api/client';
import type { HealthResponse } from '../types/health';

export function HealthStatus() {
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const loadHealth = async () => {
    setLoading(true);
    setError(null);
    try {
      setHealth(await getHealth());
    } catch (err) {
      setHealth(null);
      setError(err instanceof Error ? err.message : '无法连接后端健康检查接口');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadHealth();
  }, []);

  return (
    <Card
      title="后端健康状态"
      extra={
        <Button icon={<ReloadOutlined />} onClick={loadHealth} loading={loading}>
          刷新
        </Button>
      }
    >
      {loading && !health ? <Spin /> : null}
      {error ? <Alert type="error" showIcon message="后端未就绪" description={error} /> : null}
      {health ? (
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <Tag color={health.status === 'ok' ? 'green' : 'orange'}>{health.status}</Tag>
          <Descriptions bordered size="small" column={1}>
            <Descriptions.Item label="服务">{health.service}</Descriptions.Item>
            <Descriptions.Item label="环境">{health.environment}</Descriptions.Item>
            <Descriptions.Item label="数据库">{health.database.status}</Descriptions.Item>
            <Descriptions.Item label="已应用迁移">{health.database.appliedMigrations}</Descriptions.Item>
            <Descriptions.Item label="检查时间">{health.time}</Descriptions.Item>
          </Descriptions>
        </Space>
      ) : null}
      <Typography.Paragraph type="secondary" style={{ marginTop: 16, marginBottom: 0 }}>
        Milestone 1 仅验证应用骨架、Ant Design 接入和 /api/health 通路。
      </Typography.Paragraph>
    </Card>
  );
}
