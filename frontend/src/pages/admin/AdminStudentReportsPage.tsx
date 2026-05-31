import { Alert, Button, Card, Descriptions, Drawer, List, message, Popconfirm, Space, Table, Tag, Typography } from 'antd';
import { useEffect, useState } from 'react';
import {
  getAdminStudent,
  listAdminStudents,
  listAdminStudentWrongAnswers,
  resetStudentAttempts,
  resetStudentScore,
} from '../../api/client';
import type { StudentReportRecord, WrongAnswerRecord } from '../../types/auth';

export function AdminStudentReportsPage() {
  const [students, setStudents] = useState<StudentReportRecord[]>([]);
  const [selected, setSelected] = useState<StudentReportRecord>();
  const [wrongAnswers, setWrongAnswers] = useState<WrongAnswerRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [drawerLoading, setDrawerLoading] = useState(false);
  const [error, setError] = useState('');

  const load = () => {
    setLoading(true);
    setError('');
    listAdminStudents()
      .then(setStudents)
      .catch(() => setError('学生报告加载失败，请稍后重试。'))
      .finally(() => setLoading(false));
  };

  const openStudent = (id: number) => {
    setDrawerLoading(true);
    Promise.all([getAdminStudent(id), listAdminStudentWrongAnswers(id)])
      .then(([student, wrong]) => {
        setSelected(student);
        setWrongAnswers(wrong);
      })
      .catch(() => message.error('学生详情加载失败'))
      .finally(() => setDrawerLoading(false));
  };

  const resetAndReload = async (action: () => Promise<unknown>, text: string) => {
    await action();
    message.success(text);
    load();
    if (selected) openStudent(selected.id);
  };

  useEffect(load, []);

  return (
    <Card>
      {error && <Alert type="error" showIcon message={error} action={<Button size="small" onClick={load}>重试</Button>} />}
      <Typography.Title level={2}>学生报告与重置</Typography.Title>
      <Table
        rowKey="id"
        loading={loading}
        dataSource={students}
        locale={{ emptyText: error ? '加载失败' : '暂无学生' }}
        columns={[
          { title: '用户名', dataIndex: 'username' },
          { title: '姓名', dataIndex: 'displayName' },
          { title: '班级', render: (_, row) => row.classes?.map((item) => <Tag key={item.id}>{item.name}</Tag>) },
          { title: '总分', render: (_, row) => row.score.totalScore },
          { title: '测试', render: (_, row) => `${row.completedCount}/${row.attemptCount}` },
          { title: '状态', render: (_, row) => (row.inProgress ? <Tag color="processing">进行中</Tag> : <Tag>空闲</Tag>) },
          { title: '错题', dataIndex: 'wrongCount' },
          { title: '操作', render: (_, row) => <Button onClick={() => openStudent(row.id)}>查看</Button> },
        ]}
      />
      <Drawer width={720} open={!!selected} onClose={() => setSelected(undefined)} title={selected?.displayName} loading={drawerLoading}>
        {selected && (
          <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <Descriptions bordered size="small">
              <Descriptions.Item label="总分">{selected.score.totalScore}</Descriptions.Item>
              <Descriptions.Item label="测试次数">{selected.attemptCount}</Descriptions.Item>
              <Descriptions.Item label="错题数">{selected.wrongCount}</Descriptions.Item>
            </Descriptions>
            <Space>
              <Popconfirm title="确认清零分数并允许重新得分？" onConfirm={() => resetAndReload(() => resetStudentScore(selected.id), '分数已重置')}>
                <Button>重置分数</Button>
              </Popconfirm>
              <Popconfirm title="确认清空测试、错题和进度？" onConfirm={() => resetAndReload(() => resetStudentAttempts(selected.id), '测试记录已重置')}>
                <Button danger>重置测试记录</Button>
              </Popconfirm>
            </Space>
            <Typography.Title level={4}>测试记录</Typography.Title>
            <Table
              rowKey="id"
              size="small"
              pagination={false}
              locale={{ emptyText: '暂无测试记录' }}
              dataSource={selected.attempts || []}
              columns={[
                { title: 'ID', dataIndex: 'id' },
                { title: '状态', dataIndex: 'status' },
                { title: '答题', render: (_, row) => `${row.answeredCount}/${row.questionCount}` },
                { title: '本次得分', dataIndex: 'scoreDelta' },
                { title: '创建时间', dataIndex: 'createdAt' },
              ]}
            />
            <Typography.Title level={4}>错题</Typography.Title>
            <List
              dataSource={wrongAnswers}
              locale={{ emptyText: '暂无错题' }}
              renderItem={(item) => (
                <List.Item>
                  <List.Item.Meta
                    title={item.prompt}
                    description={`选择：${item.options[item.selectedOption]}；正确：${item.options[item.correctOption]}。${item.explanation}`}
                  />
                </List.Item>
              )}
            />
          </Space>
        )}
      </Drawer>
    </Card>
  );
}
