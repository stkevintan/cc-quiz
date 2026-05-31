import { Alert, Button, Card, List, Progress, Space, Tag, Typography, message } from 'antd';
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { getCurrentStudentAttempt, getStudentScore, listStudentWrongAnswers, startStudentAttempt } from '../../api/client';
import { useAuth } from '../../auth/AuthContext';
import type { AttemptRecord, StudentScoreRecord, WrongAnswerRecord } from '../../types/auth';

const optionLabels = ['A', 'B', 'C', 'D'];

export function StudentDashboard() {
  const { user } = useAuth();
  const navigate = useNavigate();
  const [attempt, setAttempt] = useState<AttemptRecord | null>(null);
  const [score, setScore] = useState<StudentScoreRecord | null>(null);
  const [wrongAnswers, setWrongAnswers] = useState<WrongAnswerRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  async function refreshCurrent() {
    setLoading(true);
    setError('');
    try {
      const [currentAttempt, currentScore, latestWrongAnswers] = await Promise.all([
        getCurrentStudentAttempt(),
        getStudentScore(),
        listStudentWrongAnswers(),
      ]);
      setAttempt(currentAttempt);
      setScore(currentScore);
      setWrongAnswers(latestWrongAnswers);
    } catch {
      setError('学生数据加载失败，请稍后重试。');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    refreshCurrent();
  }, []);

  async function start() {
    setLoading(true);
    try {
      const current = await startStudentAttempt();
      navigate(`/student/attempts/${current.id}`);
    } catch {
      message.error('开始测试失败，请确认题库至少有 10 道启用题目');
    } finally {
      setLoading(false);
    }
  }

  return (
    <Space direction="vertical" size="large" className="full-width">
      {error && <Alert type="error" showIcon message={error} action={<Button size="small" onClick={refreshCurrent}>重试</Button>} />}
      <Card>
        <Typography.Title level={2}>学生首页</Typography.Title>
        <Typography.Paragraph>欢迎，{user?.displayName}。每次测试固定 10 题，未完成时可刷新后继续作答。</Typography.Paragraph>
      </Card>
      <Card title="累计得分" loading={loading && !score}>
        <Typography.Title level={3}>{score?.totalScore ?? 0}</Typography.Title>
        <Typography.Text type="secondary">每道题首次答对得 1 分，重复答对不再加分。</Typography.Text>
      </Card>
      <Card title="当前测试" loading={loading && !attempt && !error}>
        {attempt ? (
          <Space direction="vertical" className="full-width">
            <Typography.Text>你有一个进行中的测试。</Typography.Text>
            <Progress percent={Math.round((attempt.answeredCount / attempt.questionCount) * 100)} />
            <Button type="primary" onClick={() => navigate(`/student/attempts/${attempt.id}`)}>
              继续作答
            </Button>
          </Space>
        ) : (
          <Space direction="vertical">
            <Typography.Text type="secondary">当前没有未完成测试。</Typography.Text>
            <Button type="primary" loading={loading} onClick={start}>
              开始 10 题测试
            </Button>
          </Space>
        )}
      </Card>
      <Card title="错题回顾">
        <List
          loading={loading && wrongAnswers.length === 0}
          locale={{ emptyText: '暂无错题记录' }}
          dataSource={wrongAnswers}
          renderItem={(item) => (
            <List.Item>
              <Space direction="vertical" className="full-width">
                <Typography.Text strong>{item.prompt}</Typography.Text>
                <Space wrap>
                  <Tag color="red">
                    上次错选：{optionLabels[item.selectedOption]}. {item.options[item.selectedOption]}
                  </Tag>
                  <Tag color="green">
                    正确答案：{optionLabels[item.correctOption]}. {item.options[item.correctOption]}
                  </Tag>
                </Space>
                <Typography.Text type="secondary">解析：{item.explanation || '暂无解析'}</Typography.Text>
              </Space>
            </List.Item>
          )}
        />
      </Card>
    </Space>
  );
}
