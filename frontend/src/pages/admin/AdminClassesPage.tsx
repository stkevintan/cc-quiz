import { Alert, Button, Card, Form, Input, Select, Space, Table, Tag, message } from 'antd';
import { useEffect, useState } from 'react';
import { bindClassMember, createClass, listClasses, listUsers } from '../../api/client';
import type { ClassRecord, ManagedUser } from '../../types/auth';

export function AdminClassesPage() {
  const [classes, setClasses] = useState<ClassRecord[]>([]);
  const [teachers, setTeachers] = useState<ManagedUser[]>([]);
  const [students, setStudents] = useState<ManagedUser[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [classForm] = Form.useForm();
  const [bindForm] = Form.useForm();

  async function refresh() {
    setLoading(true);
    setError('');
    try {
      const [nextClasses, nextTeachers, nextStudents] = await Promise.all([
        listClasses(),
        listUsers('teacher'),
        listUsers('student'),
      ]);
      setClasses(nextClasses);
      setTeachers(nextTeachers);
      setStudents(nextStudents);
    } catch {
      setError('班级或成员列表加载失败，请稍后重试。');
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
      <Card title="创建班级">
        <Form form={classForm} layout="inline" onFinish={async (values) => {
          setSubmitting(true);
          try {
            await createClass(values);
            message.success('班级已创建');
            classForm.resetFields();
            refresh();
          } catch {
            message.error('班级创建失败，请检查名称是否重复。');
          } finally {
            setSubmitting(false);
          }
        }}>
          <Form.Item name="name" rules={[{ required: true }]}><Input placeholder="班级名称" /></Form.Item>
          <Form.Item name="description"><Input placeholder="描述" /></Form.Item>
          <Button type="primary" htmlType="submit" loading={submitting}>创建</Button>
        </Form>
      </Card>
      <Card title="绑定教师/学生到班级">
        <Form form={bindForm} layout="inline" onFinish={async (values) => {
          setSubmitting(true);
          try {
            await bindClassMember(values.classId, values.role, values.userId);
            message.success('绑定已保存');
            bindForm.resetFields();
            refresh();
          } catch {
            message.error('绑定失败，请确认班级和成员仍可用。');
          } finally {
            setSubmitting(false);
          }
        }}>
          <Form.Item name="classId" rules={[{ required: true }]}><Select placeholder="班级" style={{ width: 160 }} options={classes.map((item) => ({ value: item.id, label: item.name }))} /></Form.Item>
          <Form.Item name="role" rules={[{ required: true }]}><Select placeholder="成员类型" style={{ width: 120 }} options={[{ value: 'teacher', label: '教师' }, { value: 'student', label: '学生' }]} /></Form.Item>
          <Form.Item noStyle shouldUpdate={(prev, next) => prev.role !== next.role}>
            {({ getFieldValue }) => {
              const role = getFieldValue('role');
              const options = (role === 'teacher' ? teachers : students).map((item) => ({ value: item.id, label: `${item.displayName} (${item.username})` }));
              return <Form.Item name="userId" rules={[{ required: true }]}><Select placeholder="选择成员" style={{ width: 220 }} options={options} /></Form.Item>;
            }}
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={submitting}>绑定</Button>
        </Form>
      </Card>
      <Card title="班级列表">
        <Table
          rowKey="id"
          loading={loading}
          dataSource={classes}
          locale={{ emptyText: error ? '加载失败' : '暂无班级' }}
          columns={[
            { title: '班级', dataIndex: 'name' },
            { title: '描述', dataIndex: 'description' },
            { title: '教师', render: (_, row) => row.teachers.map((item) => <Tag key={item.id}>{item.displayName}</Tag>) },
            { title: '学生', render: (_, row) => row.students.map((item) => <Tag key={item.id}>{item.displayName}</Tag>) },
          ]}
        />
      </Card>
    </Space>
  );
}
