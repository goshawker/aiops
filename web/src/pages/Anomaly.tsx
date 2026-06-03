import { Card, Form, Input, InputNumber, Select, Button, message, Tag, Space, Row, Col, Statistic, Table, Tooltip, Typography } from 'antd'
import { useState, useEffect } from 'react'
import { ThunderboltOutlined, BugOutlined, CheckCircleOutlined, CloseCircleOutlined, WarningOutlined } from '@ant-design/icons'
import { anomalyApi } from '@/api/anomaly'
import type { ModelStatus, DetectResult } from '@/api/anomaly'
import { alertsApi } from '@/api'
import type { AlertEvent } from '@/api/alerts'

const { Text } = Typography

// Color mappings for anomaly display
const typeColor: Record<string, string> = {
  '突增': 'red',
  '突降': 'blue',
  '波动': 'orange',
}

const severityColor: Record<string, string> = {
  'critical': 'red',
  'warning': 'orange',
  'info': 'blue',
}

const statusColor: Record<string, string> = {
  'open': 'red',
  'acknowledged': 'orange',
  'resolved': 'green',
  'dismissed': 'default',
}

interface AnomalyEvent {
  id: number
  metric_name: string
  anomaly_type: string
  severity: string
  value: number
  baseline: number
  confidence: number
  status: string
  detected_at: string
  feedback?: string
}

export default function Anomaly() {
  const [events, setEvents] = useState<AnomalyEvent[]>([])
  const [result, setResult] = useState<DetectResult | null>(null)
  const [detectLoading, setDetectLoading] = useState(false)
  const [modelStatus, setModelStatus] = useState<ModelStatus | null>(null)

  useEffect(() => {
    const controller = new AbortController()
    loadStatus()
    loadEvents()
    return () => controller.abort()
  }, [])

  const loadEvents = async () => {
    try {
      const res = await alertsApi.listEvents({ limit: 50 })
      const data = res.data || []
      const mapped: AnomalyEvent[] = data.map((e: AlertEvent) => ({
        id: e.id,
        metric_name: e.source,
        anomaly_type: e.title,
        severity: e.severity,
        value: Number(e.value) || 0,
        baseline: 0,
        confidence: 0,
        status: e.status,
        detected_at: e.fired_at,
        feedback: undefined,
      }))
      setEvents(mapped)
    } catch {
      message.error('加载异常事件失败')
    }
  }

  const loadStatus = async () => {
    try {
      const res = await anomalyApi.status()
      setModelStatus(res)
    } catch {
      // Anomaly service optional
    }
  }

  const handleDetect = async (values: { metric_name: string; value: number }) => {
    setDetectLoading(true)
    try {
      const res = await anomalyApi.detect(values)
      setResult(res)
    } catch { message.error('检测失败') }
    finally { setDetectLoading(false) }
  }

  const handleFeedback = (id: number, feedback: string) => {
    setEvents(events.map(e => e.id === id ? { ...e, feedback, status: 'dismissed' } : e))
    message.success(feedback === 'false_positive' ? '已标记为误报，将用于模型优化' : '已确认')
  }

  const openCount = events.filter(e => e.status === 'open').length
  const criticalCount = events.filter(e => e.severity === 'critical' && e.status !== 'resolved' && e.status !== 'dismissed').length

  const columns = [
    { title: '时间', dataIndex: 'detected_at', key: 'detected_at', width: 160 },
    { title: '指标', dataIndex: 'metric_name', key: 'metric_name', ellipsis: true, render: (m: string) => <Text code style={{ fontSize: 12 }}>{m}</Text> },
    { title: '类型', dataIndex: 'anomaly_type', key: 'anomaly_type', width: 70, render: (t: string) => <Tag color={typeColor[t]}>{t}</Tag> },
    { title: '严重度', dataIndex: 'severity', key: 'severity', width: 80, render: (s: string) => <Tag color={severityColor[s]}>{s}</Tag> },
    { title: '当前值', dataIndex: 'value', key: 'value', width: 80, render: (v: number) => <Text style={{ color: '#ff4d4f', fontWeight: 600 }}>{v.toFixed(1)}</Text> },
    { title: '基线', dataIndex: 'baseline', key: 'baseline', width: 80, render: (v: number) => <Text type="secondary">{v.toFixed(1)}</Text> },
    { title: '置信度', dataIndex: 'confidence', key: 'confidence', width: 80, render: (c: number) => <Tag>{`${(c * 100).toFixed(0)}%`}</Tag> },
    { title: '状态', dataIndex: 'status', key: 'status', width: 100, render: (s: string) => <Tag color={statusColor[s]}>{s}</Tag> },
    {
      title: '操作', key: 'action', width: 160,
      render: (_: unknown, record: AnomalyEvent) => (
        <Space>
          {record.status === 'open' && (
            <>
              <Button size="small" type="primary" onClick={() => handleFeedback(record.id, 'acknowledged')}>确认</Button>
              <Tooltip title="标记为误报，用于模型优化">
                <Button size="small" onClick={() => handleFeedback(record.id, 'false_positive')}>误报</Button>
              </Tooltip>
            </>
          )}
          {record.status === 'acknowledged' && <Tag color="orange">待处理</Tag>}
          {record.status === 'resolved' && <CheckCircleOutlined style={{ color: '#52c41a' }} />}
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        <Col span={4}>
          <Card size="small"><Statistic title="待处理" value={openCount} valueStyle={{ color: openCount > 0 ? '#ff4d4f' : '#52c41a' }} prefix={<WarningOutlined />} /></Card>
        </Col>
        <Col span={4}>
          <Card size="small"><Statistic title="严重" value={criticalCount} valueStyle={{ color: criticalCount > 0 ? '#ff4d4f' : '#52c41a' }} prefix={<CloseCircleOutlined />} /></Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Statistic
              title="检测引擎"
              value={modelStatus?.river_available ? 'River 在线学习' : '3-sigma 规则'}
              valueStyle={{ fontSize: 14, color: modelStatus?.river_available ? '#52c41a' : '#faad14' }}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Space wrap size={[4, 4]}>
              <Statistic title="活跃模型" value={modelStatus?.model_count || 0} valueStyle={{ fontSize: 14 }} />
              {modelStatus && Object.entries(modelStatus.models).slice(0, 3).map(([key, info]) => (
                <Tag key={key} color={info.warmup_done ? 'green' : 'orange'} style={{ marginTop: 8 }}>
                  {key.split('{')[0]}
                </Tag>
              ))}
            </Space>
          </Card>
        </Col>
      </Row>

      <Card title={<Space><BugOutlined />异常事件</Space>} size="small" style={{ marginBottom: 16, padding: 12 }}>
        <Table
          dataSource={events}
          columns={columns}
          rowKey="id"
          size="small"
          virtual
          scroll={{ x: 1000, y: 500 }}
          pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 条` }}
        />
      </Card>

      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={12}>
          <Card title="手动检测" size="small">
            <Form layout="inline" onFinish={handleDetect}>
              <Form.Item name="metric_name" label="指标名" rules={[{ required: true }]}>
                <Input placeholder="cpu_usage" style={{ width: 160 }} />
              </Form.Item>
              <Form.Item name="value" label="值" rules={[{ required: true }]}>
                <InputNumber style={{ width: 100 }} />
              </Form.Item>
              <Form.Item>
                <Button type="primary" htmlType="submit" loading={detectLoading} icon={<ThunderboltOutlined />}>检测</Button>
              </Form.Item>
            </Form>
            {result && (
              <div style={{ marginTop: 12 }}>
                {result.anomaly ? (
                  <Space><Tag color="red" style={{ fontSize: 13, padding: '2px 8px' }}>异常</Tag><Text code>{JSON.stringify(result.result)}</Text></Space>
                ) : (
                  <Tag color="green" style={{ fontSize: 13, padding: '2px 8px' }}>正常</Tag>
                )}
              </div>
            )}
          </Card>
        </Col>
        <Col span={12}>
          <Card title="设置阈值规则" size="small">
            <Form layout="inline" onFinish={async (values) => {
              try { await anomalyApi.setThreshold(values); message.success('阈值已设置') }
              catch { message.error('设置失败') }
            }}>
              <Form.Item name="metric_name" label="指标" rules={[{ required: true }]}>
                <Input placeholder="cpu_usage" style={{ width: 140 }} />
              </Form.Item>
              <Form.Item name="op" label="条件" initialValue=">">
                <Select style={{ width: 70 }} options={[{ label: '>', value: '>' }, { label: '>=', value: '>=' }, { label: '<', value: '<' }, { label: '<=', value: '<=' }]} />
              </Form.Item>
              <Form.Item name="value" label="阈值">
                <InputNumber style={{ width: 100 }} />
              </Form.Item>
              <Form.Item>
                <Button type="primary" htmlType="submit">设置</Button>
              </Form.Item>
            </Form>
          </Card>
        </Col>
      </Row>
    </div>
  )
}
