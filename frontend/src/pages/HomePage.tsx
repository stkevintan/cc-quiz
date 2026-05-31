import { Card, Col, Row, Typography } from 'antd';
import { HealthStatus } from '../components/HealthStatus';

export function HomePage() {
  return (
    <Row gutter={[24, 24]}>
      <Col xs={24} lg={12}>
        <Card>
          <Typography.Title level={2}>项目骨架已就绪</Typography.Title>
          <Typography.Paragraph>
            本阶段搭建 Go + Gin + SQLite 后端，以及 React + TypeScript + Ant Design 前端。
          </Typography.Paragraph>
          <Typography.Paragraph>
            登录、角色、题目管理、答题和积分流程将在后续里程碑实现。
          </Typography.Paragraph>
        </Card>
      </Col>
      <Col xs={24} lg={12}>
        <HealthStatus />
      </Col>
    </Row>
  );
}
