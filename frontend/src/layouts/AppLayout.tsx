import { Button, Layout, Space, Typography } from 'antd';
import type { PropsWithChildren } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';

const { Header, Content, Footer } = Layout;

export function AppLayout({ children }: PropsWithChildren) {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  return (
    <Layout className="app-layout">
      <Header className="app-header">
        <Link to={user ? `/${user.role}` : '/login'}>
          <Typography.Title level={3} className="app-title">
            文言文选择题
          </Typography.Title>
        </Link>
        <Space className="app-nav">
          {user && (
            <>
              {user.role === 'admin' && (
                <>
                  <Link to="/admin/users">账号</Link>
                  <Link to="/admin/classes">班级</Link>
                  <Link to="/admin/questions">题库</Link>
                  <Link to="/admin/reports">报告</Link>
                </>
              )}
              {user.role === 'teacher' && (
                <>
                  <Link to="/teacher/students">学生</Link>
                  <Link to="/teacher/questions">题库</Link>
                </>
              )}
              {user.role === 'student' && <Link to="/student">测试</Link>}
              <span>
                {user.displayName}（{user.role}）
              </span>
              <Button
                onClick={() => {
                  logout();
                  navigate('/login');
                }}
              >
                登出
              </Button>
            </>
          )}
        </Space>
      </Header>
      <Content className="app-content">{children}</Content>
      <Footer className="app-footer">Classical Chinese Quiz · Milestone 6</Footer>
    </Layout>
  );
}
