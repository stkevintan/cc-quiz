import { Card, Col, Row, Typography } from 'antd';
import { Link } from 'react-router-dom';

export function AdminDashboard() {
  return (
    <Row gutter={[16, 16]}>
      <Col xs={24} md={12}>
        <Link to="/admin/users">
          <Card hoverable>
            <Typography.Title level={3}>账号管理</Typography.Title>
            创建/停用教师和学生账号，重置密码。
          </Card>
        </Link>
      </Col>
      <Col xs={24} md={12}>
        <Link to="/admin/classes">
          <Card hoverable>
            <Typography.Title level={3}>班级管理</Typography.Title>
            创建班级并维护教师、学生绑定关系。
          </Card>
        </Link>
      </Col>
      <Col xs={24} md={12}>
        <Link to="/admin/questions">
          <Card hoverable>
            <Typography.Title level={3}>题库管理</Typography.Title>
            维护文言文选择题、选项、答案和解析。
          </Card>
        </Link>
      </Col>
      <Col xs={24} md={12}>
        <Link to="/admin/reports">
          <Card hoverable>
            <Typography.Title level={3}>学生报告</Typography.Title>
            查看分数、测试记录、错题并执行重置。
          </Card>
        </Link>
      </Col>
    </Row>
  );
}
