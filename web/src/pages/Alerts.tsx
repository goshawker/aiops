import { useEffect, useState } from 'react'
import { Card, Table, Tag, Button, Space, Tabs, Typography, message, Select, Row, Col, Statistic } from 'antd'
import {
  CheckOutlined,
  CheckCircleOutlined,
  AlertOutlined,
  CloseCircleOutlined,
  WarningOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import { alertsApi } from '@/api'
import type { AlertEvent, Incident } from '@/api/alerts'

const { Title } = Typography

const severityColor = (s: string) => {
  if (s === 'critical') return 'red'
  if (s === 'warning') return 'orange'
  return 'blue'
}

const statusColor = (s: string) => {
  if (s === 'open') return 'red'
  if (s === 'acknowledged') return 'orange'
  if (s === 'resolved' || s === 'closed') return 'green'
  return 'default'
}

export default function Alerts() {
  const [events, setEvents] = useState<AlertEvent[]>([])
  const [incidents, setIncidents] = useState<Incident[]>([])
  const [loading, setLoading] = useState(false)
  const [tab, setTab] = useState('incidents')
  const [severityFilter, setSeverityFilter] = useState<string>('')
  const [statusFilter, setStatusFilter] = useState<string>('')

  useEffect(() => {
    loadData()
  }, [tab])

  const loadData = async () => {
    setLoading(true)
    try {
      if (tab === 'incidents') {
        const res = await alertsApi.listIncidents({ limit: 200 })
        setIncidents(res.data || [])
      } else {
        const res = await alertsApi.listEvents({ limit: 200 })
        setEvents(res.data || [])
      }
    } catch (e: any) {
      message.error(e?.error || '加载告警数据失败')
    } finally {
      setLoading(false)
    }
  }

  const handleAcknowledge = async (id: number) => {
    try {
      await alertsApi.acknowledge(id)
      message.success('已确认')
      loadData()
    } catch (e) {
      message.error('操作失败')
    }
  }

  const handleResolve = async (id: number) => {
    try {
      await alertsApi.resolve(id)
      message.success('已解决')
      loadData()
    } catch (e) {
      message.error('操作失败')
    }
  }

  // Apply filters
  const filteredIncidents = incidents.filter((i) => {
    if (severityFilter && i.severity !== severityFilter) return false
    if (statusFilter && i.status !== statusFilter) return false
    return true
  })

  const filteredEvents = events.filter((e) => {
    if (severityFilter && e.severity !== severityFilter) return false
    return true
  })

  // Stats
  const criticalCount = incidents.filter((i) => i.severity === 'critical').length
  const warningCount = incidents.filter((i) => i.severity === 'warning').length
  const openCount = incidents.filter((i) => i.status === 'open').length

  const incidentColumns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 70 },
    {
      title: '严重程度',
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      render: (s: string) => <Tag color={severityColor(s)}>{s}</Tag>,
    },
    { title: '标题', dataIndex: 'title', key: 'title', ellipsis: true },
    {
      title: '影响服务',
      dataIndex: 'affected_services',
      key: 'affected_services',
      width: 150,
      render: (s: string[]) =>
        s?.length ? s.map((svc) => <Tag key={svc}>{svc}</Tag>) : '-',
    },
    { title: '事件数', dataIndex: 'event_count', key: 'event_count', width: 70 },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 110,
      render: (s: string) => <Tag color={statusColor(s)}>{s}</Tag>,
    },
    { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
    {
      title: '操作',
      key: 'action',
      width: 140,
      render: (_: any, record: Incident) => (
        <Space>
          {record.status === 'open' && (
            <Button size="small" icon={<CheckOutlined />} onClick={() => handleAcknowledge(record.id)}>
              确认
            </Button>
          )}
          {record.status !== 'resolved' && record.status !== 'closed' && (
            <Button
              size="small"
              type="primary"
              icon={<CheckCircleOutlined />}
              onClick={() => handleResolve(record.id)}
            >
              解决
            </Button>
          )}
        </Space>
      ),
    },
  ]

  const eventColumns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 70 },
    {
      title: '严重程度',
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      render: (s: string) => <Tag color={severityColor(s)}>{s}</Tag>,
    },
    { title: '标题', dataIndex: 'title', key: 'title', ellipsis: true },
    { title: '来源', dataIndex: 'source', key: 'source', width: 120 },
    { title: '主机', dataIndex: 'host', key: 'host', width: 120 },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (s: string) => <Tag color={s === 'firing' ? 'red' : 'green'}>{s}</Tag>,
    },
    { title: '触发时间', dataIndex: 'fired_at', key: 'fired_at', width: 170 },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>
          <AlertOutlined /> 告警管理
        </Title>
        <Button icon={<ReloadOutlined />} onClick={loadData} loading={loading}>
          刷新
        </Button>
      </div>

      {/* Stats row */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="活跃事件"
              value={incidents.length}
              prefix={<AlertOutlined />}
              valueStyle={{ color: openCount > 0 ? '#faad14' : '#52c41a' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="严重"
              value={criticalCount}
              prefix={<CloseCircleOutlined />}
              valueStyle={{ color: '#ff4d4f' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="警告"
              value={warningCount}
              prefix={<WarningOutlined />}
              valueStyle={{ color: '#faad14' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="待处理"
              value={openCount}
              valueStyle={{ color: openCount > 0 ? '#ff4d4f' : '#52c41a' }}
            />
          </Card>
        </Col>
      </Row>

      {/* Filters */}
      <Card size="small" style={{ marginBottom: 16 }}>
        <Space wrap>
          <Select
            placeholder="严重程度"
            value={severityFilter || undefined}
            onChange={(v) => setSeverityFilter(v || '')}
            allowClear
            style={{ width: 120 }}
            options={[
              { label: 'critical', value: 'critical' },
              { label: 'warning', value: 'warning' },
              { label: 'info', value: 'info' },
            ]}
          />
          {tab === 'incidents' && (
            <Select
              placeholder="状态"
              value={statusFilter || undefined}
              onChange={(v) => setStatusFilter(v || '')}
              allowClear
              style={{ width: 120 }}
              options={[
                { label: 'open', value: 'open' },
                { label: 'acknowledged', value: 'acknowledged' },
                { label: 'resolved', value: 'resolved' },
                { label: 'closed', value: 'closed' },
              ]}
            />
          )}
        </Space>
      </Card>

      {/* Tabs */}
      <Card>
        <Tabs
          activeKey={tab}
          onChange={setTab}
          items={[
            {
              key: 'incidents',
              label: `事件 (${filteredIncidents.length})`,
              children: (
                <Table
                  dataSource={filteredIncidents}
                  columns={incidentColumns}
                  rowKey="id"
                  loading={loading}
                  size="small"
                  scroll={{ x: 900 }}
                  pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 条` }}
                />
              ),
            },
            {
              key: 'events',
              label: `告警 (${filteredEvents.length})`,
              children: (
                <Table
                  dataSource={filteredEvents}
                  columns={eventColumns}
                  rowKey="id"
                  loading={loading}
                  size="small"
                  scroll={{ x: 900 }}
                  pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 条` }}
                />
              ),
            },
          ]}
        />
      </Card>
    </div>
  )
}
