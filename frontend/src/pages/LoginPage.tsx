import { Alert, Button, Card, Form, Input, Typography } from 'antd';
import { useState } from 'react';
import { Navigate, useNavigate } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';

export function LoginPage() {
  const { user, login } = useAuth();
  const navigate = useNavigate();
  const [error, setError] = useState('');

  if (user) {
    return <Navigate to={`/${user.role}`} replace />;
  }

  return (
    <Card className="login-card">
      <Typography.Title level={2}>登录</Typography.Title>
      <Typography.Paragraph type="secondary">
        本地默认账号：admin/admin123、teacher1/teacher123、student1/student123
      </Typography.Paragraph>
      {error && <Alert type="error" message={error} showIcon className="form-alert" />}
      <Form
        layout="vertical"
        onFinish={async (values) => {
          setError('');
          try {
            const loggedIn = await login(values.username, values.password);
            navigate(`/${loggedIn.role}`, { replace: true });
          } catch {
            setError('用户名或密码错误');
          }
        }}
      >
        <Form.Item name="username" label="用户名" rules={[{ required: true }]}>
          <Input autoComplete="username" />
        </Form.Item>
        <Form.Item name="password" label="密码" rules={[{ required: true }]}>
          <Input.Password autoComplete="current-password" />
        </Form.Item>
        <Button type="primary" htmlType="submit" block>
          登录
        </Button>
      </Form>
    </Card>
  );
}
