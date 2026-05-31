import { Alert, Button, Card, Progress, Radio, Space, Tag, Typography, message } from 'antd';
import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { getStudentAttempt, submitStudentAttemptAnswer } from '../../api/client';
import type { AttemptRecord } from '../../types/auth';

const optionLabels = ['A', 'B', 'C', 'D'];

export function StudentAttemptPage() {
  const { attemptId } = useParams();
  const navigate = useNavigate();
  const [attempt, setAttempt] = useState<AttemptRecord | null>(null);
  const [currentIndex, setCurrentIndex] = useState(0);
  const [selectedOption, setSelectedOption] = useState<number | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const numericAttemptId = Number(attemptId);

  useEffect(() => {
    async function load() {
      if (!numericAttemptId) {
        navigate('/student');
        return;
      }
      setLoading(true);
      setError('');
      try {
        const loaded = await getStudentAttempt(numericAttemptId);
        setAttempt(loaded);
        const next = loaded.questions.findIndex((question) => !question.answer);
        setCurrentIndex(next >= 0 ? next : loaded.questions.length - 1);
      } catch {
        setError('无法加载测试，请返回学生首页后重试。');
      } finally {
        setLoading(false);
      }
    }
    load();
  }, [numericAttemptId, navigate]);

  const currentQuestion = attempt?.questions[currentIndex];
  const progressPercent = useMemo(() => {
    if (!attempt) return 0;
    return Math.round((attempt.answeredCount / attempt.questionCount) * 100);
  }, [attempt]);

  async function submitAnswer() {
    if (!attempt || !currentQuestion || selectedOption === null) {
      message.warning('请选择一个答案');
      return;
    }
    setLoading(true);
    try {
      const result = await submitStudentAttemptAnswer(attempt.id, {
        attemptQuestionId: currentQuestion.id,
        selectedOption,
      });
      setAttempt(result.attempt);
      setSelectedOption(null);
      if (result.answer.isCorrect) {
        message.success(result.answer.scoreDelta === 1 ? '回答正确，获得 1 分' : '回答正确，本题此前已得分');
      } else {
        message.info('答案已提交，已更新错题记录');
      }
    } catch {
      setError('提交失败，可能本题已作答或网络异常。');
      message.error('提交失败，可能本题已作答');
    } finally {
      setLoading(false);
    }
  }

  function goNext() {
    if (!attempt) return;
    const next = attempt.questions.findIndex((question, index) => index > currentIndex && !question.answer);
    if (next >= 0) {
      setCurrentIndex(next);
      return;
    }
    if (attempt.status === 'completed') {
      navigate('/student');
    }
  }

  if (!attempt || !currentQuestion) {
    return (
      <Card loading={loading}>
        {error ? (
          <Alert
            type="error"
            showIcon
            message={error}
            action={<Button size="small" onClick={() => navigate('/student')}>返回首页</Button>}
          />
        ) : '加载中'}
      </Card>
    );
  }

  const answer = currentQuestion.answer;
  const completed = attempt.status === 'completed';
  const correctOption = currentQuestion.correctOption ?? 0;

  return (
    <Space direction="vertical" size="large" className="full-width">
      {error && <Alert type="error" showIcon message={error} closable onClose={() => setError('')} />}
      <Card>
        <Space direction="vertical" className="full-width">
          <Space className="full-width" style={{ justifyContent: 'space-between' }}>
            <Typography.Title level={2}>10 题测试</Typography.Title>
            <Link to="/student">返回学生首页</Link>
          </Space>
          <Progress percent={progressPercent} />
          <Typography.Text>
            已答 {attempt.answeredCount} / {attempt.questionCount}
            {completed && <Tag color="green" style={{ marginLeft: 8 }}>已完成</Tag>}
          </Typography.Text>
          <Typography.Text>本次新增得分：{attempt.scoreDelta}</Typography.Text>
        </Space>
      </Card>

      <Card
        title={`第 ${currentIndex + 1} 题`}
        extra={answer ? <Tag color={answer.isCorrect ? 'green' : 'red'}>{answer.isCorrect ? '正确' : '错误'}</Tag> : <Tag>未作答</Tag>}
      >
        <Space direction="vertical" size="large" className="full-width">
          <Typography.Paragraph>{currentQuestion.prompt}</Typography.Paragraph>
          <Radio.Group
            className="full-width"
            disabled={Boolean(answer) || completed}
            value={answer ? answer.selectedOption : selectedOption}
            onChange={(event) => setSelectedOption(event.target.value)}
          >
            <Space direction="vertical">
              {currentQuestion.options.map((option, index) => (
                <Radio key={`${currentQuestion.id}-${index}`} value={index}>
                  {optionLabels[index]}. {option}
                </Radio>
              ))}
            </Space>
          </Radio.Group>

          {answer && (
            <Alert
              type={answer.isCorrect ? 'success' : 'error'}
              showIcon
              message={answer.isCorrect ? '回答正确' : '回答错误'}
              description={
                <Space direction="vertical">
                  <span>
                    正确答案：{optionLabels[correctOption]}. {currentQuestion.options[correctOption]}
                  </span>
                  <span>本题新增得分：{answer.scoreDelta}</span>
                  <span>解析：{currentQuestion.explanation || '暂无解析'}</span>
                </Space>
              }
            />
          )}

          <Space>
            {!answer && !completed && (
              <Button type="primary" loading={loading} onClick={submitAnswer}>
                提交答案
              </Button>
            )}
            {answer && !completed && (
              <Button type="primary" onClick={goNext}>
                下一题
              </Button>
            )}
            {completed && (
              <Button type="primary" onClick={() => navigate('/student')}>
                完成，返回首页
              </Button>
            )}
          </Space>
        </Space>
      </Card>
    </Space>
  );
}
