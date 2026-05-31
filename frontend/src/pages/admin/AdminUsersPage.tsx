import { Alert, Button, Card, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, message } from 'antd';
import { useEffect, useState } from 'react';
import { createUser, disableUser, listClasses, listUsers, resetPassword } from '../../api/client';
import type { ClassRecord, ManagedUser } from '../../types/auth';

export function AdminUsersPage() {
  const [users, setUsers] = useState<ManagedUser[]>([]);
  const [classes, setClasses] = useState<ClassRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [form] = Form.useForm();

  async function refresh() {
    setLoading(true);
    setError('');
    try {
      const [nextUsers, nextClasses] = await Promise.all([listUsers(), listClasses()]);
      setUsers(nextUsers);
      setClasses(nextClasses);
    } catch {
      setError('账号或班级加载失败，请稍后重试。');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    refresh();
  }, []);

  return (
    <Space direction="vertical" size="large" className="full-width">
      {error && <Alert type="error" showIcon message={error} action={<Button size="small" onClick={refresh}>重试</Button>} />}
      <Card title="创建教师/学生账号">
        <Form
          form={form}
          layout="inline"
          onFinish={async (values) => {
            setSubmitting(true);
            try {
              await createUser({ ...values, classIds: values.classIds || [] });
              message.success('账号已创建');
              form.resetFields();
              refresh();
            } catch {
              message.error('账号创建失败，请检查用户名是否重复。');
            } finally {
              setSubmitting(false);
            }
          }}
        >
          <Form.Item name="username" rules={[{ required: true }]}><Input placeholder="用户名" /></Form.Item>
          <Form.Item name="displayName" rules={[{ required: true }]}><Input placeholder="显示名称" /></Form.Item>
          <Form.Item name="password" rules={[{ required: true, min: 6 }]}><Input.Password placeholder="初始密码" /></Form.Item>
          <Form.Item name="role" rules={[{ required: true }]}><Select placeholder="角色" style={{ width: 120 }} options={[{ value: 'teacher', label: '教师' }, { value: 'student', label: '学生' }]} /></Form.Item>
          <Form.Item name="classIds"><Select mode="multiple" placeholder="绑定班级" style={{ minWidth: 180 }} options={classes.map((item) => ({ value: item.id, label: item.name }))} /></Form.Item>
          <Button type="primary" htmlType="submit" loading={submitting}>创建</Button>
        </Form>
      </Card>
      <Card title="账号列表">
        <Table
          rowKey="id"
          loading={loading}
          dataSource={users}
          locale={{ emptyText: error ? '加载失败' : '暂无账号' }}
          columns={[
            { title: '用户名', dataIndex: 'username' },
            { title: '姓名', dataIndex: 'displayName' },
            { title: '角色', dataIndex: 'role', render: (role) => <Tag>{role}</Tag> },
            { title: '状态', dataIndex: 'status' },
            { title: '班级', render: (_, row) => row.classes?.map((item) => item.name).join('、') || '-' },
            {
              title: '操作',
              render: (_, row) => (
                <Space>
                  <Button
                    onClick={() => {
                      Modal.confirm({
                        title: `重置 ${row.displayName} 的密码`,
                        content: '密码将重置为 ChangeMe123',
                        onOk: async () => {
                          await resetPassword(row.id, 'ChangeMe123');
                          message.success('密码已重置为 ChangeMe123');
                        },
                      });
                    }}
                  >
                    重置密码
                  </Button>
                  <Popconfirm
                    title={`确认停用 ${row.displayName}？`}
                    description="停用后该账号将无法登录。"
                    okText="确认停用"
                    cancelText="取消"
                    onConfirm={async () => {
                      try {
                        await disableUser(row.id);
                        message.success('账号已停用');
                        refresh();
                      } catch {
                        message.error('账号停用失败');
                      }
                    }}
                  >
                    <Button danger disabled={row.status === 'disabled'}>
                      停用
                    </Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Card>
    </Space>
  );
}
