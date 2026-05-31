import { Alert, Button, Card, Descriptions, Drawer, List, Space, Table, Tag, Typography, message } from 'antd';
import { useEffect, useState } from 'react';
import { getTeacherStudent, listTeacherStudents, listTeacherStudentWrongAnswers } from '../../api/client';
import type { StudentReportRecord, WrongAnswerRecord } from '../../types/auth';

export function TeacherStudentsPage() {
  const [students, setStudents] = useState<StudentReportRecord[]>([]);
  const [selected, setSelected] = useState<StudentReportRecord>();
  const [wrongAnswers, setWrongAnswers] = useState<WrongAnswerRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [drawerLoading, setDrawerLoading] = useState(false);
  const [error, setError] = useState('');

  const load = () => {
    setLoading(true);
    setError('');
    listTeacherStudents()
      .then(setStudents)
      .catch(() => setError('学生列表加载失败，请稍后重试。'))
      .finally(() => setLoading(false));
  };

  useEffect(load, []);

  const openStudent = (id: number) => {
    setDrawerLoading(true);
    Promise.all([getTeacherStudent(id), listTeacherStudentWrongAnswers(id)])
      .then(([student, wrong]) => {
        setSelected(student);
        setWrongAnswers(wrong);
      })
      .catch(() => message.error('学生详情加载失败'))
      .finally(() => setDrawerLoading(false));
  };

  return (
    <Card>
      {error && <Alert type="error" showIcon message={error} action={<Button size="small" onClick={load}>重试</Button>} />}
      <Typography.Title level={2}>我的班级学生</Typography.Title>
      <Typography.Paragraph type="secondary">后端仅返回当前教师绑定班级内的学生。</Typography.Paragraph>
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
          { title: '操作', render: (_, row) => <a onClick={() => openStudent(row.id)}>查看</a> },
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
