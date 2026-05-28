import { useEffect, useState } from 'react'
import { Card, Row, Col, Statistic, Tag, List, Typography, Progress, Space, Badge } from 'antd'
import {
  AlertOutlined,
  CheckCircleOutlined,
  WarningOutlined,
  CloseCircleOutlined,
  DashboardOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import { alertsApi, llmApi } from '@/api'
import type { Incident } from '@/api/alerts'
import type { SummaryResponse as LlmSummary } from '@/api/llm'

const { Title, Text, Paragraph } = Typography

// Mock data for charts (will be replaced with real API calls)
const mockAlertTrend = {
  hours: Array.from({ length: 24 }, (_, i) => `${i}:00`),
  critical: [0, 0, 1, 0, 0, 2, 1, 0, 0, 1, 3, 2, 1, 0, 0, 1, 2, 1, 0, 0, 1, 0, 0, 0],
  warning: [2, 1, 3, 2, 1, 4, 3, 2, 1, 3, 5, 4, 3, 2, 1, 3, 4, 3, 2, 1, 3, 2, 1, 2],
  info: [5, 4, 6, 5, 4, 8, 7, 5, 4, 6, 10, 8, 6, 5, 4, 6, 8, 6, 5, 4, 6, 5, 4, 5],
}

const mockResourceUsage = {
  cpu: 45,
  memory: 62,
  disk: 78,
  network: 23,
}

export default function Dashboard() {
  const [incidents, setIncidents] = useState<Incident[]>([])
  const [summary, setSummary] = useState<LlmSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [healthScore, setHealthScore] = useState(85)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const [incidentsRes, summaryRes] = await Promise.allSettled([
        alertsApi.listIncidents({ status: 'open', limit: 10 }),
        llmApi.summary({
          metrics: [
            { name: 'cpu_usage', value: mockResourceUsage.cpu },
            { name: 'memory_usage', value: mockResourceUsage.memory },
            { name: 'disk_usage', value: mockResourceUsage.disk },
          ],
        }),
      ])

      if (incidentsRes.status === 'fulfilled') {
        setIncidents(incidentsRes.value.data || [])
      }
      if (summaryRes.status === 'fulfilled') {
        setSummary(summaryRes.value)
        // Calculate health score based on status
        if (summaryRes.value.status === 'critical') setHealthScore(30)
        else if (summaryRes.value.status === 'warning') setHealthScore(65)
        else setHealthScore(95)
      }
    } catch (e) {
      console.error('Failed to load dashboard:', e)
    } finally {
      setLoading(false)
    }
  }

  const severityIcon = (s: string) => {
    if (s === 'critical') return <CloseCircleOutlined style={{ color: '#ff4d4f', fontSize: 16 }} />
    if (s === 'warning') return <WarningOutlined style={{ color: '#faad14', fontSize: 16 }} />
    return <CheckCircleOutlined style={{ color: '#52c41a', fontSize: 16 }} />
  }

  const severityColor = (s: string) => {
    if (s === 'critical') return 'red'
    if (s === 'warning') return 'orange'
    return 'green'
  }

  const healthColor = (score: number) => {
    if (score >= 80) return '#52c41a'
    if (score >= 60) return '#faad14'
    return '#ff4d4f'
  }

  // Alert trend chart option
  const alertTrendOption = {
    tooltip: { trigger: 'axis' },
    legend: { data: ['严重', '警告', '信息'], bottom: 0 },
    grid: { left: '3%', right: '4%', bottom: '15%', top: '5%', containLabel: true },
    xAxis: {
      type: 'category',
      data: mockAlertTrend.hours,
      axisLabel: { fontSize: 10 },
    },
    yAxis: { type: 'value', name: '告警数' },
    series: [
      {
        name: '严重',
        type: 'bar',
        stack: 'total',
        data: mockAlertTrend.critical,
        itemStyle: { color: '#ff4d4f' },
      },
      {
        name: '警告',
        type: 'bar',
        stack: 'total',
        data: mockAlertTrend.warning,
        itemStyle: { color: '#faad14' },
      },
      {
        name: '信息',
        type: 'bar',
        stack: 'total',
        data: mockAlertTrend.info,
        itemStyle: { color: '#1677ff' },
      },
    ],
  }

  // Resource usage chart option
  const resourceOption = {
    tooltip: { trigger: 'item' },
    radar: {
      indicator: [
        { name: 'CPU', max: 100 },
        { name: '内存', max: 100 },
        { name: '磁盘', max: 100 },
        { name: '网络', max: 100 },
      ],
    },
    series: [
      {
        type: 'radar',
        data: [
          {
            value: [mockResourceUsage.cpu, mockResourceUsage.memory, mockResourceUsage.disk, mockResourceUsage.network],
            name: '资源使用率',
            areaStyle: { color: 'rgba(22, 119, 255, 0.2)' },
            lineStyle: { color: '#1677ff' },
          },
        ],
      },
    ],
  }

  // Severity distribution pie chart
  const severityDistribution = {
    tooltip: { trigger: 'item' },
    series: [
      {
        type: 'pie',
        radius: ['40%', '70%'],
        avoidLabelOverlap: false,
        itemStyle: { borderRadius: 10, borderColor: '#fff', borderWidth: 2 },
        label: { show: false },
        emphasis: { label: { show: true, fontSize: 14, fontWeight: 'bold' } },
        data: [
          { value: incidents.filter((i) => i.severity === 'critical').length, name: '严重', itemStyle: { color: '#ff4d4f' } },
          { value: incidents.filter((i) => i.severity === 'warning').length, name: '警告', itemStyle: { color: '#faad14' } },
          { value: incidents.filter((i) => i.severity === 'info').length, name: '信息', itemStyle: { color: '#1677ff' } },
        ].filter((d) => d.value > 0),
      },
    ],
  }

  return (
    <div>
      <Title level={4} style={{ marginBottom: 24 }}>
        <DashboardOutlined /> 仪表盘总览
      </Title>

      {/* Top row: Health score + key metrics */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <div style={{ textAlign: 'center' }}>
              <Progress
                type="dashboard"
                percent={healthScore}
                strokeColor={healthColor(healthScore)}
                format={(percent) => (
                  <div>
                    <div style={{ fontSize: 28, fontWeight: 700, color: healthColor(percent!) }}>{percent}</div>
                    <div style={{ fontSize: 12, color: '#999' }}>健康分</div>
                  </div>
                )}
              />
              <div style={{ marginTop: 8 }}>
                <Tag color={healthScore >= 80 ? 'green' : healthScore >= 60 ? 'orange' : 'red'}>
                  {healthScore >= 80 ? '系统正常' : healthScore >= 60 ? '需要关注' : '存在异常'}
                </Tag>
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="活跃事件"
              value={incidents.length}
              prefix={<AlertOutlined />}
              valueStyle={{ color: incidents.length > 0 ? '#faad14' : '#52c41a' }}
            />
            <div style={{ marginTop: 8, fontSize: 12, color: '#999' }}>
              严重 {incidents.filter((i) => i.severity === 'critical').length} · 警告{' '}
              {incidents.filter((i) => i.severity === 'warning').length}
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="CPU 使用率" value={mockResourceUsage.cpu} suffix="%" valueStyle={{ color: mockResourceUsage.cpu > 80 ? '#ff4d4f' : '#1677ff' }} />
            <Progress percent={mockResourceUsage.cpu} showInfo={false} strokeColor={mockResourceUsage.cpu > 80 ? '#ff4d4f' : '#1677ff'} size="small" />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="内存使用率" value={mockResourceUsage.memory} suffix="%" valueStyle={{ color: mockResourceUsage.memory > 85 ? '#ff4d4f' : '#1677ff' }} />
            <Progress percent={mockResourceUsage.memory} showInfo={false} strokeColor={mockResourceUsage.memory > 85 ? '#ff4d4f' : '#1677ff'} size="small" />
          </Card>
        </Col>
      </Row>

      {/* AI Health Summary */}
      {summary && (
        <Card
          title={
            <Space>
              <span>AI 健康摘要</span>
              <Tag color={severityColor(summary.status)}>{summary.status.toUpperCase()}</Tag>
              <Tag>{summary.source}</Tag>
            </Space>
          }
          style={{ marginBottom: 24 }}
        >
          <Paragraph style={{ fontSize: 16, marginBottom: 16 }}>{summary.summary}</Paragraph>
          {summary.recommendations.length > 0 && (
            <div>
              <Text strong>建议：</Text>
              <ul style={{ marginTop: 8, paddingLeft: 20 }}>
                {summary.recommendations.map((r: string, i: number) => (
                  <li key={i}>{r}</li>
                ))}
              </ul>
            </div>
          )}
        </Card>
      )}

      {/* Charts row */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} lg={16}>
          <Card title="24 小时告警趋势" size="small">
            <ReactECharts option={alertTrendOption} style={{ height: 300 }} />
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="资源使用率" size="small">
            <ReactECharts option={resourceOption} style={{ height: 300 }} />
          </Card>
        </Col>
      </Row>

      {/* Bottom row: Incidents + Severity distribution */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={16}>
          <Card
            title="活跃事件列表"
            size="small"
            extra={<Badge count={incidents.length} style={{ backgroundColor: '#ff4d4f' }} />}
          >
            <List
              loading={loading}
              dataSource={incidents}
              locale={{ emptyText: '暂无活跃事件' }}
              renderItem={(item: Incident) => (
                <List.Item
                  actions={[
                    <Tag key="status" color={severityColor(item.severity)}>
                      {item.severity}
                    </Tag>,
                  ]}
                >
                  <List.Item.Meta
                    avatar={severityIcon(item.severity)}
                    title={item.title}
                    description={
                      <Space>
                        <span>{item.affected_services?.join(', ') || '-'}</span>
                        <span>·</span>
                        <span>{item.event_count} 条事件</span>
                        <span>·</span>
                        <ClockCircleOutlined />
                        <span>{item.created_at}</span>
                      </Space>
                    }
                  />
                </List.Item>
              )}
            />
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="事件严重程度分布" size="small">
            {incidents.length > 0 ? (
              <ReactECharts option={severityDistribution} style={{ height: 250 }} />
            ) : (
              <div style={{ height: 250, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#999' }}>
                暂无数据
              </div>
            )}
          </Card>
        </Col>
      </Row>
    </div>
  )
}
