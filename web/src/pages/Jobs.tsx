import { useState, useEffect, useRef } from 'react'
import { Card, Table, Tag, Button, Space, Typography, message, Modal, Form, Input, Select, InputNumber } from 'antd'
import { PlusOutlined, PlayCircleOutlined, HistoryOutlined, DeleteOutlined, ClockCircleOutlined } from '@ant-design/icons'
import client from '@/api/client'

const { Text } = Typography

interface Job {
  id: number
  name: string
  description: string
  job_type: string
  content: string
  schedule: string
  enabled: boolean
  status: string
  timeout: number
  last_run_at: string | null
}

interface JobExecution {
  id: number
  job_id: number
  status: string
  output: string
  error: string
  duration: number
  started_at: string
  ended_at: string | null
}

export default function Jobs() {
  const [jobs, setJobs] = useState<Job[]>([])
  const [loading, setLoading] = useState(false)
  const [createVisible, setCreateVisible] = useState(false)
  const [historyVisible, setHistoryVisible] = useState(false)
  const [executions, setExecutions] = useState<JobExecution[]>([])
  const [selectedJob, setSelectedJob] = useState<Job | null>(null)
  const [form] = Form.useForm()
  const refreshTimer = useRef<ReturnType<typeof setTimeout>>()

  useEffect(() => {
    loadJobs()
    return () => { if (refreshTimer.current) clearTimeout(refreshTimer.current) }
  }, [])

  const loadJobs = async () => {
    setLoading(true)
    try {
      const res = await client.get('/jobs', { params: { limit: 100 } })
      setJobs(res.data?.data || [])
    } catch (e) {
      message.error('加载作业列表失败')
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (values: any) => {
    try {
      await client.post('/jobs', values)
      message.success('作业已创建')
      setCreateVisible(false)
      form.resetFields()
      loadJobs()
    } catch (e) {
      message.error('创建失败')
    }
  }

  const handleRun = async (job: Job) => {
    try {
      await client.post(`/jobs/${job.id}/run`)
      message.success(`作业 "${job.name}" 已开始执行`)
      refreshTimer.current = setTimeout(loadJobs, 1000)
    } catch (e) {
      message.error('执行失败')
    }
  }

  const handleDelete = async (job: Job) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除作业 "${job.name}" 吗？`,
      onOk: async () => {
        try {
          await client.delete(`/jobs/${job.id}`)
          message.success('已删除')
          loadJobs()
        } catch (e) {
          message.error('删除失败')
        }
      },
    })
  }

  const loadHistory = async (job: Job) => {
    setSelectedJob(job)
    try {
      const res = await client.get(`/jobs/${job.id}/executions`)
      setExecutions(res.data?.data || [])
      setHistoryVisible(true)
    } catch (e) {
      message.error('加载历史失败')
    }
  }

  const statusColor = (s: string) => {
    if (s === 'success') return 'green'
    if (s === 'failed') return 'red'
    if (s === 'running') return 'blue'
    return 'default'
  }

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      width: 150,
      render: (name: string, record: Job) => (
        <Space>
          <Text strong>{name}</Text>
          {!record.enabled && <Tag>已禁用</Tag>}
        </Space>
      ),
    },
    {
      title: '类型',
      dataIndex: 'job_type',
      key: 'job_type',
      width: 80,
      render: (t: string) => <Tag>{t}</Tag>,
    },
    {
      title: '命令/URL',
      dataIndex: 'content',
      key: 'content',
      ellipsis: true,
    },
    {
      title: '调度',
      dataIndex: 'schedule',
      key: 'schedule',
      width: 100,
      render: (s: string) => (
        <Space>
          <ClockCircleOutlined />
          <Text code>{s}</Text>
        </Space>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (s: string) => <Tag color={statusColor(s)}>{s}</Tag>,
    },
    {
      title: '上次运行',
      dataIndex: 'last_run_at',
      key: 'last_run_at',
      width: 160,
      render: (t: string | null) => t ? new Date(t).toLocaleString('zh-CN') : '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      render: (_: any, record: Job) => (
        <Space>
          <Button size="small" type="primary" icon={<PlayCircleOutlined />} onClick={() => handleRun(record)}>
            执行
          </Button>
          <Button size="small" icon={<HistoryOutlined />} onClick={() => loadHistory(record)}>
            历史
          </Button>
          <Button size="small" danger icon={<DeleteOutlined />} onClick={() => handleDelete(record)} />
        </Space>
      ),
    },
  ]

  const execColumns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (s: string) => <Tag color={statusColor(s)}>{s}</Tag>,
    },
    {
      title: '耗时',
      dataIndex: 'duration',
      key: 'duration',
      width: 100,
      render: (ms: number) => ms ? `${(ms / 1000).toFixed(1)}s` : '-',
    },
    {
      title: '输出',
      dataIndex: 'output',
      key: 'output',
      ellipsis: true,
    },
    {
      title: '错误',
      dataIndex: 'error',
      key: 'error',
      ellipsis: true,
      render: (e: string) => e ? <Text type="danger">{e}</Text> : '-',
    },
    {
      title: '开始时间',
      dataIndex: 'started_at',
      key: 'started_at',
      width: 160,
      render: (t: string) => new Date(t).toLocaleString('zh-CN'),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>
          新建作业
        </Button>
      </div>

      <Card style={{ padding: 16 }}>
        <Table
          dataSource={jobs}
          columns={columns}
          rowKey="id"
          loading={loading}
          size="small"
          virtual
          scroll={{ x: 1000, y: 500 }}
          pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 个` }}
        />
      </Card>

      {/* Create modal */}
      <Modal
        title="新建作业"
        open={createVisible}
        onCancel={() => setCreateVisible(false)}
        onOk={() => form.submit()}
      >
        <Form form={form} layout="vertical" onFinish={handleCreate}>
          <Form.Item name="name" label="作业名称" rules={[{ required: true }]}>
            <Input placeholder="清理日志" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input placeholder="清理 30 天前的日志文件" />
          </Form.Item>
          <Form.Item name="job_type" label="类型" rules={[{ required: true }]}>
            <Select options={[
              { label: 'Shell 命令', value: 'shell' },
              { label: 'HTTP 检查', value: 'http' },
            ]} />
          </Form.Item>
          <Form.Item name="content" label="命令/URL" rules={[{ required: true }]}>
            <Input.TextArea rows={3} placeholder="find /var/log -name '*.log' -mtime +30 -delete" />
          </Form.Item>
          <Form.Item name="schedule" label="调度" initialValue="once">
            <Select options={[
              { label: '手动执行', value: 'once' },
              { label: '每小时', value: '0 * * * *' },
              { label: '每天', value: '0 2 * * *' },
              { label: '每周', value: '0 2 * * 1' },
            ]} />
          </Form.Item>
          <Form.Item name="timeout" label="超时（秒）" initialValue={300}>
            <InputNumber min={10} max={3600} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>

      {/* History modal */}
      <Modal
        title={`执行历史: ${selectedJob?.name}`}
        open={historyVisible}
        onCancel={() => setHistoryVisible(false)}
        footer={null}
        width={900}
      >
        <Table
          dataSource={executions}
          columns={execColumns}
          rowKey="id"
          size="small"
          pagination={{ pageSize: 10 }}
        />
      </Modal>
    </div>
  )
}
