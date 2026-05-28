import { useState, useEffect } from 'react'
import { Card, Table, Tag, Space, Typography, Select, message, Tooltip } from 'antd'
import { BranchesOutlined } from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import client from '@/api/client'

const { Title, Text } = Typography

interface TraceSummary {
  trace_id: string
  root_service: string
  root_operation: string
  span_count: number
  duration_ms: number
  status_code: string
  start_time: string
}

interface Span {
  timestamp: string
  trace_id: string
  span_id: string
  parent_span_id: string
  service: string
  operation: string
  duration_ms: number
  status_code: string
  attributes: Record<string, string>
}

export default function Traces() {
  const [traces, setTraces] = useState<TraceSummary[]>([])
  const [selectedTrace, setSelectedTrace] = useState<Span[]>([])
  const [selectedTraceID, setSelectedTraceID] = useState<string>('')
  const [services, setServices] = useState<string[]>([])
  const [serviceFilter, setServiceFilter] = useState<string>('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    loadTraces()
    loadServices()
  }, [serviceFilter])

  const loadTraces = async () => {
    setLoading(true)
    try {
      const params: any = { limit: 100 }
      if (serviceFilter) params.service = serviceFilter
      const res = await client.get('/traces', { params })
      setTraces(res.data?.data || [])
    } catch (e) {
      // Service may not be available
    } finally {
      setLoading(false)
    }
  }

  const loadServices = async () => {
    try {
      const res = await client.get('/traces/services')
      setServices(res.data?.data || [])
    } catch (e) {
      // Ignore
    }
  }

  const loadTraceDetail = async (traceID: string) => {
    try {
      const res = await client.get(`/traces/${traceID}`)
      setSelectedTrace(res.data?.data || [])
      setSelectedTraceID(traceID)
    } catch (e) {
      message.error('加载链路详情失败')
    }
  }

  // Build Gantt chart option for trace detail
  const getGanttOption = () => {
    if (selectedTrace.length === 0) return null

    const services = [...new Set(selectedTrace.map((s) => s.service))]
    const serviceColors: Record<string, string> = {}
    const colors = ['#1677ff', '#52c41a', '#faad14', '#ff4d4f', '#722ed1', '#13c2c2', '#eb2f96']
    services.forEach((s, i) => {
      serviceColors[s] = colors[i % colors.length]
    })

    const minTime = Math.min(...selectedTrace.map((s) => new Date(s.timestamp).getTime()))

    const data = selectedTrace.map((span) => {
      const start = new Date(span.timestamp).getTime() - minTime
      const end = start + span.duration_ms
      return {
        name: `${span.service}: ${span.operation}`,
        value: [start, end],
        itemStyle: {
          color: serviceColors[span.service],
        },
        span,
      }
    })

    return {
      tooltip: {
        formatter: (params: any) => {
          const span = params.data.span
          return `<div style="font-size:12px">
            <b>${span.service}: ${span.operation}</b><br/>
            Span ID: ${span.span_id}<br/>
            Duration: ${span.duration_ms.toFixed(2)}ms<br/>
            Status: ${span.status_code}
          </div>`
        },
      },
      grid: { left: '15%', right: '4%', top: '5%', bottom: '10%' },
      xAxis: {
        type: 'value',
        name: 'ms',
        axisLabel: {
          formatter: (val: number) => val.toFixed(0),
        },
      },
      yAxis: {
        type: 'category',
        data: data.map((d) => d.name),
        axisLabel: { fontSize: 11 },
      },
      series: [
        {
          type: 'custom',
          renderItem: (_params: any, api: any) => {
            const start = api.coord([api.value(0), 0])
            const end = api.coord([api.value(1), 0])
            const height = api.size([0, 1])[1] * 0.6

            return {
              type: 'rect',
              shape: {
                x: start[0],
                y: start[1] - height / 2,
                width: end[0] - start[0],
                height: height,
              },
              style: api.style(),
            }
          },
          encode: {
            x: [0, 1],
          },
          data: data,
        },
      ],
    }
  }

  const traceColumns = [
    {
      title: 'Trace ID',
      dataIndex: 'trace_id',
      key: 'trace_id',
      width: 200,
      render: (id: string) => (
        <Tooltip title={id}>
          <Tag
            style={{ cursor: 'pointer' }}
            onClick={() => loadTraceDetail(id)}
          >
            {id.substring(0, 16)}...
          </Tag>
        </Tooltip>
      ),
    },
    {
      title: '服务',
      dataIndex: 'root_service',
      key: 'root_service',
      width: 120,
    },
    {
      title: '操作',
      dataIndex: 'root_operation',
      key: 'root_operation',
      ellipsis: true,
    },
    {
      title: 'Spans',
      dataIndex: 'span_count',
      key: 'span_count',
      width: 70,
    },
    {
      title: '耗时',
      dataIndex: 'duration_ms',
      key: 'duration_ms',
      width: 100,
      render: (ms: number) => (
        <Text code style={{ color: ms > 1000 ? '#ff4d4f' : ms > 500 ? '#faad14' : '#52c41a' }}>
          {ms.toFixed(1)}ms
        </Text>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status_code',
      key: 'status_code',
      width: 80,
      render: (s: string) => <Tag color={s === 'ERROR' ? 'red' : 'green'}>{s}</Tag>,
    },
    {
      title: '时间',
      dataIndex: 'start_time',
      key: 'start_time',
      width: 170,
      render: (t: string) => new Date(t).toLocaleString('zh-CN'),
    },
  ]

  return (
    <div>
      <Title level={4}>
        <BranchesOutlined /> 链路追踪
      </Title>

      {/* Filters */}
      <Card size="small" style={{ marginBottom: 16 }}>
        <Space>
          <Select
            placeholder="服务名称"
            value={serviceFilter || undefined}
            onChange={(v) => setServiceFilter(v || '')}
            allowClear
            showSearch
            style={{ width: 200 }}
            options={services.map((s) => ({ label: s, value: s }))}
          />
        </Space>
      </Card>

      {/* Trace list */}
      <Card title={`链路列表 (${traces.length})`} size="small" style={{ marginBottom: 16 }}>
        <Table
          dataSource={traces}
          columns={traceColumns}
          rowKey="trace_id"
          loading={loading}
          size="small"
          scroll={{ x: 900 }}
          pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 条` }}
          onRow={(record) => ({
            onClick: () => loadTraceDetail(record.trace_id),
            style: { cursor: 'pointer' },
          })}
        />
      </Card>

      {/* Trace detail - Gantt chart */}
      {selectedTrace.length > 0 && (
        <Card
          title={`链路详情: ${selectedTraceID.substring(0, 16)}... (${selectedTrace.length} spans)`}
          size="small"
        >
          <ReactECharts option={getGanttOption()} style={{ height: Math.max(selectedTrace.length * 30 + 80, 200) }} />
        </Card>
      )}
    </div>
  )
}
