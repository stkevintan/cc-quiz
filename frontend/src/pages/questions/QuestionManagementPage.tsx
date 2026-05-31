import { Alert, Button, Card, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, Typography, message } from 'antd';
import { useEffect, useMemo, useState } from 'react';
import { createQuestion, disableQuestion, listQuestions, updateQuestion } from '../../api/client';
import type { QuestionRecord } from '../../types/auth';

const optionLabels = ['A', 'B', 'C', 'D'];
const defaultOptions = ['', '', '', ''];

interface QuestionFormValues {
  prompt: string;
  options: string[];
  correctOption: number;
  explanation: string;
}

export function QuestionManagementPage() {
  const [questions, setQuestions] = useState<QuestionRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<QuestionRecord | null>(null);
  const [form] = Form.useForm<QuestionFormValues>();

  async function refresh() {
    setLoading(true);
    setError('');
    try {
      setQuestions(await listQuestions());
    } catch {
      setError('题库加载失败，请稍后重试。');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    refresh();
  }, []);

  const modalTitle = useMemo(() => (editing ? '编辑题目' : '创建题目'), [editing]);

  function openCreate() {
    setEditing(null);
    form.setFieldsValue({ prompt: '', options: defaultOptions, correctOption: 0, explanation: '' });
    setModalOpen(true);
  }

  function openEdit(question: QuestionRecord) {
    setEditing(question);
    form.setFieldsValue({
      prompt: question.prompt,
      options: [...question.options, ...defaultOptions].slice(0, 4),
      correctOption: question.correctOption,
      explanation: question.explanation,
    });
    setModalOpen(true);
  }

  async function submit(values: QuestionFormValues) {
    const options = (values.options || []).map((item) => item?.trim() || '').filter(Boolean);
    if (options.length < 2 || options.length > 4) {
      message.error('选项数量必须为 2 到 4 个');
      return;
    }
    if (!values.options?.[values.correctOption]?.trim()) {
      message.error('正确答案必须指向已填写的选项');
      return;
    }
    const correctOption = values.options.slice(0, values.correctOption).filter((item) => item?.trim()).length;
    const payload = {
      prompt: values.prompt.trim(),
      options,
      correctOption,
      explanation: values.explanation?.trim() || '',
    };
    setSubmitting(true);
    try {
      if (editing) {
        await updateQuestion(editing.id, payload);
        message.success('题目已更新');
      } else {
        await createQuestion(payload);
        message.success('题目已创建');
      }
      setModalOpen(false);
      form.resetFields();
      refresh();
    } catch {
      message.error('题目保存失败，请检查题干和选项。');
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Space direction="vertical" size="large" className="full-width">
      {error && <Alert type="error" showIcon message={error} action={<Button size="small" onClick={refresh}>重试</Button>} />}
      <Card>
        <Space className="full-width" direction="vertical">
          <Space className="full-width" style={{ justifyContent: 'space-between' }}>
            <div>
              <Typography.Title level={2}>题库管理</Typography.Title>
              <Typography.Paragraph type="secondary">
                管理文言文选择题。每题支持 2-4 个选项，创建新题默认提供 4 个可编辑选项。
              </Typography.Paragraph>
            </div>
            <Button type="primary" onClick={openCreate}>创建题目</Button>
          </Space>
          <Table
            rowKey="id"
            loading={loading}
            dataSource={questions}
            locale={{ emptyText: error ? '加载失败' : '暂无题目' }}
            columns={[
              { title: '题干', dataIndex: 'prompt', ellipsis: true },
              { title: '选项数', dataIndex: 'optionCount', width: 90 },
              {
                title: '答案',
                width: 90,
                render: (_, row) => <Tag color="green">{optionLabels[row.correctOption]}</Tag>,
              },
              {
                title: '状态',
                width: 100,
                render: (_, row) => <Tag color={row.status === 'active' ? 'blue' : 'default'}>{row.status}</Tag>,
              },
              {
                title: '操作',
                width: 180,
                render: (_, row) => (
                  <Space>
                    <Button onClick={() => openEdit(row)}>编辑</Button>
                    <Popconfirm
                      title="确认停用该题目？"
                      description="停用后新测试不会再抽取该题。"
                      okText="确认停用"
                      cancelText="取消"
                      onConfirm={async () => {
                        try {
                          await disableQuestion(row.id);
                          message.success('题目已停用');
                          refresh();
                        } catch {
                          message.error('题目停用失败');
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
            expandable={{
              expandedRowRender: (row) => (
                <Space direction="vertical">
                  {row.options.map((option, index) => (
                    <span key={optionLabels[index]}>
                      {optionLabels[index]}. {option}
                    </span>
                  ))}
                  <Typography.Text type="secondary">解析：{row.explanation || '无'}</Typography.Text>
                </Space>
              ),
            }}
          />
        </Space>
      </Card>
      <Modal
        title={modalTitle}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={submitting}
        destroyOnHidden
        width={720}
      >
        <Form form={form} layout="vertical" onFinish={submit} initialValues={{ options: defaultOptions, correctOption: 0 }}>
          <Form.Item name="prompt" label="题干" rules={[{ required: true, message: '请输入题干' }]}>
            <Input.TextArea rows={3} placeholder="输入文言文选择题题干" />
          </Form.Item>
          {optionLabels.map((label, index) => (
            <Form.Item key={label} name={['options', index]} label={`选项 ${label}`}>
              <Input placeholder={index < 2 ? '至少填写前两个选项' : '可选'} />
            </Form.Item>
          ))}
          <Form.Item name="correctOption" label="正确答案" rules={[{ required: true, message: '请选择正确答案' }]}>
            <Select options={optionLabels.map((label, index) => ({ value: index, label: `选项 ${label}` }))} />
          </Form.Item>
          <Form.Item name="explanation" label="解析">
            <Input.TextArea rows={3} placeholder="可填写答案解析，供后续练习与复盘使用" />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
