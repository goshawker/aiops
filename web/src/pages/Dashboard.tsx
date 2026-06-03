import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, Row, Col, Tag, List, Typography, Progress, Space, Button, message } from 'antd'
import {
  AlertOutlined,
  DashboardOutlined,
  ClockCircleOutlined,
  ThunderboltOutlined,
  ApartmentOutlined,
  MessageOutlined,
  HddOutlined,
  FundOutlined,
} from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import { alertsApi, llmApi, metricsApi } from '@/api'
import type { Incident, AlertEvent } from '@/api/alerts'
import type { SummaryResponse as LlmSummary } from '@/api/llm'
import { useAppStore } from '@/store/app'

const { Text, Paragraph } = Typography

export default function Dashboard() {
  const [incidents, setIncidents] = useState<Incident[]>([])
  const [summary, setSummary] = useState<LlmSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [healthScore, setHealthScore] = useState(85)
  const [resourceUsage, setResourceUsage] = useState({ cpu: 0, memory: 0, disk: 0, network: 0 })
  const [alertTrend, setAlertTrend] = useState<{
    hours: string[]; critical: number[]; warning: number[]; info: number[]
  }>({
    hours: Array.from({ length: 24 }, (_, i) => `${i}:00`),
    critical: Array(24).fill(0),
    warning: Array(24).fill(0),
    info: Array(24).fill(0),
  })
  const { setAssistantVisible } = useAppStore()
  const navigate = useNavigate()

  useEffect(() => { loadData() }, [])

  const loadData = async () => {
    try {
      const [incidentsRes, eventsRes, cpuRes, memRes, diskRes] = await Promise.allSettled([
        alertsApi.listIncidents({ status: 'open', limit: 10 }),
        alertsApi.listEvents({ limit: 200 }),
        metricsApi.instant('100 - (avg(rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)'),
        metricsApi.instant('(1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100'),
        metricsApi.instant('(1 - node_filesystem_avail_bytes{mountpoint="/"} / node_filesystem_size_bytes{mountpoint="/"}) * 100'),
      ])

      // Incidents
      if (incidentsRes.status === 'fulfilled') setIncidents(incidentsRes.value.data || [])

      // Resource usage from metrics
      const extractValue = (res: PromiseSettledResult<{ data?: { values?: { value: number }[] }[] }>): number => {
        if (res.status !== 'fulfilled') return 0
        const data = res.value?.data
        if (!data || data.length === 0) return 0
        const lastVal = data[0]?.values
        return lastVal ? Math.round(lastVal[lastVal.length - 1]?.value || 0) : 0
      }

      const cpu = extractValue(cpuRes)
      const mem = extractValue(memRes)
      const disk = extractValue(diskRes)
      setResourceUsage({ cpu, memory: mem, disk, network: 0 })

      // Build 24h alert trend from events
      if (eventsRes.status === 'fulfilled') {
        const events: AlertEvent[] = eventsRes.value.data || []
        const hours = Array.from({ length: 24 }, (_, i) => `${i}:00`)
        const critical = Array(24).fill(0)
        const warning = Array(24).fill(0)
        const info = Array(24).fill(0)

        events.forEach((e) => {
          const h = new Date(e.fired_at).getHours()
          if (e.severity === 'critical') critical[h]++
          else if (e.severity === 'warning') warning[h]++
          else info[h]++
        })

        setAlertTrend({ hours, critical, warning, info })
      }

      // LLM summary with real metrics
      try {
        const summaryRes = await llmApi.summary({
          metrics: [
            { name: 'cpu_usage', value: cpu },
            { name: 'memory_usage', value: mem },
            { name: 'disk_usage', value: disk },
          ],
        })
        setSummary(summaryRes)
        if (summaryRes.status === 'critical') setHealthScore(30)
        else if (summaryRes.status === 'warning') setHealthScore(65)
        else setHealthScore(95)
      } catch {
        // LLM may not be available
        setHealthScore(cpu > 90 || mem > 95 ? 30 : cpu > 70 || mem > 80 ? 65 : 95)
      }
    } catch (e) {
      message.error('加载仪表盘数据失败')
    } finally { setLoading(false) }
  }

  const healthColor = (score: number) => {
    if (score >= 80) return '#52c41a'
    if (score >= 60) return '#faad14'
    return '#ff4d4f'
  }

  const criticalCount = incidents.filter((i) => i.severity === 'critical').length
  const warningCount = incidents.filter((i) => i.severity === 'warning').length

  // ── Chart Options ──────────────────────────────────────
  const alertTrendOption = {
    tooltip: { trigger: 'axis' },
    legend: { data: ['严重', '警告', '信息'], bottom: 0, icon: 'circle', itemWidth: 8 },
    grid: { left: '3%', right: '4%', bottom: '18%', top: '5%', containLabel: true },
    xAxis: { type: 'category', data: alertTrend.hours, axisLabel: { fontSize: 10 }, axisLine: { show: false }, axisTick: { show: false } },
    yAxis: { type: 'value', name: '告警数', axisLabel: { fontSize: 10 }, splitLine: { lineStyle: { type: 'dashed', color: '#e8e8e8' } } },
    series: [
      { name: '严重', type: 'bar', stack: 'total', barMaxWidth: 20, data: alertTrend.critical, itemStyle: { color: '#ff4d4f', borderRadius: [0, 0, 0, 0] } },
      { name: '警告', type: 'bar', stack: 'total', barMaxWidth: 20, data: alertTrend.warning, itemStyle: { color: '#faad14' } },
      { name: '信息', type: 'bar', stack: 'total', barMaxWidth: 20, data: alertTrend.info, itemStyle: { color: '#1677ff', borderRadius: [2, 2, 0, 0] } },
    ],
  }

  const resourceOption = {
    tooltip: { trigger: 'item' },
    radar: {
      indicator: [
        { name: 'CPU', max: 100 }, { name: '内存', max: 100 },
        { name: '磁盘', max: 100 }, { name: '网络', max: 100 },
      ],
      shape: 'circle', splitNumber: 4, axisName: { color: '#888' },
      splitLine: { lineStyle: { color: 'rgba(0,0,0,0.1)' } },
      splitArea: { areaStyle: { color: ['rgba(22,119,255,0.02)', 'rgba(22,119,255,0.02)'] } },
    },
    series: [{ type: 'radar', data: [{ value: [resourceUsage.cpu, resourceUsage.memory, resourceUsage.disk, resourceUsage.network], name: '资源使用率', areaStyle: { color: 'rgba(22, 119, 255, 0.15)' }, lineStyle: { color: '#1677ff', width: 2 }, itemStyle: { color: '#1677ff' } }] }],
  }

  const severityDistribution = {
    tooltip: { trigger: 'item' },
    series: [{
      type: 'pie', radius: ['45%', '70%'], avoidLabelOverlap: false,
      itemStyle: { borderRadius: 6, borderColor: '#fff', borderWidth: 3 },
      label: { show: false },
      emphasis: { label: { show: true, fontSize: 13, fontWeight: 'bold' } },
      data: [
        { value: Math.max(incidents.filter((i) => i.severity === 'critical').length, 1), name: '严重', itemStyle: { color: '#ff4d4f' } },
        { value: Math.max(incidents.filter((i) => i.severity === 'warning').length, 2), name: '警告', itemStyle: { color: '#faad14' } },
        { value: Math.max(incidents.filter((i) => i.severity === 'info').length, 3), name: '信息', itemStyle: { color: '#1677ff' } },
      ],
    }],
  }

  return (
    <div>
      {/* Stats Card Row */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card style={{ padding: 20 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
              <div style={{ width: 56, height: 56, borderRadius: 12, background: `linear-gradient(135deg, ${healthColor(healthScore)}22, ${healthColor(healthScore)}44)`, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <DashboardOutlined style={{ fontSize: 24, color: healthColor(healthScore) }} />
              </div>
              <div>
                <Text type="secondary" style={{ fontSize: 13 }}>系统健康分</Text>
                <div style={{ fontSize: 28, fontWeight: 700, color: healthColor(healthScore), lineHeight: 1.2 }}>{healthScore}</div>
                <Tag color={healthScore >= 80 ? 'green' : healthScore >= 60 ? 'orange' : 'red'} style={{ marginTop: 2 }}>
                  {healthScore >= 80 ? '正常' : healthScore >= 60 ? '关注' : '异常'}
                </Tag>
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card style={{ padding: 20 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
              <div style={{ width: 56, height: 56, borderRadius: 12, background: incidents.length > 0 ? '#fff2f0' : '#f6ffed', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <AlertOutlined style={{ fontSize: 24, color: incidents.length > 0 ? '#ff4d4f' : '#52c41a' }} />
              </div>
              <div>
                <Text type="secondary" style={{ fontSize: 13 }}>活跃事件</Text>
                <div style={{ fontSize: 28, fontWeight: 700, color: incidents.length > 0 ? '#ff4d4f' : '#52c41a', lineHeight: 1.2 }}>{incidents.length}</div>
                <Space size={4} style={{ marginTop: 2 }}>
                  <span style={{ fontSize: 12, color: '#ff4d4f' }}>{criticalCount} 严重</span>
                  <span style={{ fontSize: 12, color: '#999' }}>·</span>
                  <span style={{ fontSize: 12, color: '#faad14' }}>{warningCount} 警告</span>
                </Space>
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card style={{ padding: 20 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
              <div style={{ width: 56, height: 56, borderRadius: 12, background: resourceUsage.cpu > 80 ? '#fff2f0' : '#e6f4ff', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <FundOutlined style={{ fontSize: 24, color: resourceUsage.cpu > 80 ? '#ff4d4f' : '#1677ff' }} />
              </div>
              <div style={{ flex: 1 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Text type="secondary" style={{ fontSize: 13 }}>CPU 使用率</Text>
                  <Text style={{ fontSize: 16, fontWeight: 600, color: resourceUsage.cpu > 80 ? '#ff4d4f' : '#333' }}>{resourceUsage.cpu}%</Text>
                </div>
                <Progress percent={resourceUsage.cpu} showInfo={false} strokeColor={resourceUsage.cpu > 80 ? '#ff4d4f' : '#1677ff'} size="small" style={{ marginTop: 4 }} />
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card style={{ padding: 20 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
              <div style={{ width: 56, height: 56, borderRadius: 12, background: resourceUsage.memory > 85 ? '#fff2f0' : '#e6f4ff', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <HddOutlined style={{ fontSize: 24, color: resourceUsage.memory > 85 ? '#ff4d4f' : '#1677ff' }} />
              </div>
              <div style={{ flex: 1 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Text type="secondary" style={{ fontSize: 13 }}>内存使用率</Text>
                  <Text style={{ fontSize: 16, fontWeight: 600, color: resourceUsage.memory > 85 ? '#ff4d4f' : '#333' }}>{resourceUsage.memory}%</Text>
                </div>
                <Progress percent={resourceUsage.memory} showInfo={false} strokeColor={resourceUsage.memory > 85 ? '#ff4d4f' : '#1677ff'} size="small" style={{ marginTop: 4 }} />
              </div>
            </div>
          </Card>
        </Col>
      </Row>

      {/* AI Health Summary */}
      {summary && (
        <Card title={<Space><ThunderboltOutlined style={{ color: '#1677ff' }} /><span>AI 健康摘要</span><Tag color={summary.status === 'critical' ? 'red' : summary.status === 'warning' ? 'orange' : 'green'}>{summary.status.toUpperCase()}</Tag></Space>} style={{ marginBottom: 24 }} styles={{ body: { padding: 20 } }}>
          <Paragraph style={{ fontSize: 15, marginBottom: 16 }}>{summary.summary}</Paragraph>
          {summary.recommendations && summary.recommendations.length > 0 && (
            <div>
              <Text strong style={{ fontSize: 13 }}>建议操作：</Text>
              <div style={{ marginTop: 8, display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                {summary.recommendations.map((r: string, i: number) => (
                  <Tag key={i} style={{ padding: '4px 12px', fontSize: 13, cursor: 'pointer' }}>{r}</Tag>
                ))}
              </div>
            </div>
          )}
        </Card>
      )}

      {/* Charts */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} lg={16}>
          <Card title="24 小时告警趋势" size="small" styles={{ body: { padding: 12 } }}>
            <ReactECharts option={alertTrendOption} style={{ height: 280 }} />
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="资源使用率" size="small" styles={{ body: { padding: 12 } }}>
            <ReactECharts option={resourceOption} style={{ height: 280 }} />
          </Card>
        </Col>
      </Row>

      {/* Events + Quick Actions */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={16}>
          <Card title={<Space><AlertOutlined /><span>活跃事件</span><Tag color="red">{incidents.length}</Tag></Space>} size="small">
            <List
              loading={loading}
              dataSource={incidents}
              locale={{ emptyText: '暂无活跃事件，系统运行正常' }}
              size="small"
              renderItem={(item: Incident) => {
                const sevColor = item.severity === 'critical' ? '#ff4d4f' : item.severity === 'warning' ? '#faad14' : '#1677ff'
                return (
                  <List.Item style={{ padding: '10px 0' }} actions={[<Tag key="sev" color={sevColor}>{item.severity}</Tag>]}>
                    <List.Item.Meta
                      avatar={<div style={{ width: 8, height: 8, borderRadius: '50%', background: sevColor, marginTop: 8 }} />}
                      title={<Text style={{ fontSize: 14 }}>{item.title}</Text>}
                      description={
                        <Space size={12}>
                          <Text type="secondary" style={{ fontSize: 12 }}>{(item.affected_services || []).join(', ') || '-'}</Text>
                          <Text type="secondary" style={{ fontSize: 12 }}>{item.event_count || 0} 条事件</Text>
                          <Text type="secondary" style={{ fontSize: 12 }}><ClockCircleOutlined /> {item.created_at}</Text>
                        </Space>
                      }
                    />
                  </List.Item>
                )
              }}
            />
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Space direction="vertical" style={{ width: '100%' }} size={16}>
            <Card title="事件分布" size="small" styles={{ body: { padding: 12 } }}>
              <ReactECharts option={severityDistribution} style={{ height: 200 }} />
            </Card>
            <Card title="快捷操作" size="small" styles={{ body: { padding: 16 } }}>
              <Space direction="vertical" style={{ width: '100%' }} size={12}>
                <Button block icon={<ApartmentOutlined />} onClick={() => navigate('/rca')}>根因分析</Button>
                <Button block icon={<MessageOutlined />} onClick={() => setAssistantVisible(true)}>AI 助手</Button>
                <Button block icon={<ThunderboltOutlined />}>执行作业</Button>
              </Space>
            </Card>
          </Space>
        </Col>
      </Row>
    </div>
  )
}
