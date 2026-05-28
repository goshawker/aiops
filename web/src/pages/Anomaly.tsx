import { Card, Typography, Form, Input, InputNumber, Select, Button, message, Tag, Space, Row, Col, Statistic } from 'antd'
import { useState, useEffect } from 'react'
import { BugOutlined, ThunderboltOutlined } from '@ant-design/icons'
import client from '@/api/client'

const { Title } = Typography

interface ModelStatus {
  river_available: boolean
  model_count: number
  models: Record<string, {
    point_count: number
    warmup_done: boolean
    has_river_model: boolean
    recent_score: number
  }>
}

export default function Anomaly() {
  const [result, setResult] = useState<any>(null)
  const [loading, setLoading] = useState(false)
  const [modelStatus, setModelStatus] = useState<ModelStatus | null>(null)

  useEffect(() => {
    loadStatus()
  }, [])

  const loadStatus = async () => {
    try {
      const res = await client.get('/anomaly/status')
      setModelStatus(res.data)
    } catch (e) {
      // Service may not be running
    }
  }

  const handleDetect = async (values: any) => {
    setLoading(true)
    try {
      const res = await client.post('/anomaly/detect', values)
      setResult(res)
    } catch (e) {
      message.error('检测失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <Title level={4}>
        <BugOutlined /> 异常检测
      </Title>

      {/* Model status */}
      {modelStatus && (
        <Card size="small" style={{ marginBottom: 16 }}>
          <Row gutter={16}>
            <Col span={6}>
              <Statistic
                title="检测引擎"
                value={modelStatus.river_available ? 'River 在线学习' : '3-sigma 规则'}
                valueStyle={{ fontSize: 14, color: modelStatus.river_available ? '#52c41a' : '#faad14' }}
              />
            </Col>
            <Col span={6}>
              <Statistic title="活跃模型" value={modelStatus.model_count} />
            </Col>
            <Col span={12}>
              <Space wrap size={[4, 4]}>
                {Object.entries(modelStatus.models).slice(0, 5).map(([key, info]) => (
                  <Tag key={key} color={info.warmup_done ? 'green' : 'orange'}>
                    {key.split('{')[0]}: {info.point_count} pts
                  </Tag>
                ))}
              </Space>
            </Col>
          </Row>
        </Card>
      )}

      {/* Manual detection */}
      <Card title="手动检测" style={{ marginBottom: 16 }}>
        <Form layout="inline" onFinish={handleDetect}>
          <Form.Item name="metric_name" label="指标名" rules={[{ required: true }]}>
            <Input placeholder="cpu_usage" />
          </Form.Item>
          <Form.Item name="value" label="值" rules={[{ required: true }]}>
            <InputNumber style={{ width: 120 }} />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} icon={<ThunderboltOutlined />}>
              检测
            </Button>
          </Form.Item>
        </Form>
      </Card>

      {/* Detection result */}
      {result && (
        <Card title="检测结果" style={{ marginBottom: 16 }}>
          {result.anomaly ? (
            <Space direction="vertical" style={{ width: '100%' }}>
              <Tag color="red" style={{ fontSize: 14, padding: '4px 12px' }}>
                异常 detected
              </Tag>
              <pre style={{ background: '#f5f5f5', padding: 12, borderRadius: 4, fontSize: 12 }}>
                {JSON.stringify(result.result, null, 2)}
              </pre>
            </Space>
          ) : (
            <Tag color="green" style={{ fontSize: 14, padding: '4px 12px' }}>
              正常 — 未检测到异常
            </Tag>
          )}
        </Card>
      )}

      {/* Threshold setting */}
      <Card title="设置阈值规则">
        <Form
          layout="inline"
          onFinish={async (values) => {
            try {
              await client.post('/anomaly/thresholds', values)
              message.success('阈值已设置')
            } catch (e) {
              message.error('设置失败')
            }
          }}
        >
          <Form.Item name="metric_name" label="指标名" rules={[{ required: true }]}>
            <Input placeholder="cpu_usage" />
          </Form.Item>
          <Form.Item name="op" label="运算符" initialValue=">">
            <Select
              style={{ width: 80 }}
              options={[
                { label: '>', value: '>' },
                { label: '>=', value: '>=' },
                { label: '<', value: '<' },
                { label: '<=', value: '<=' },
              ]}
            />
          </Form.Item>
          <Form.Item name="value" label="阈值" rules={[{ required: true }]}>
            <InputNumber style={{ width: 120 }} />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit">
              设置
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  )
}
