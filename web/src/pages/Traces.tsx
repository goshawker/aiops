import { useState, useEffect, useMemo } from 'react'
import { Card, Table, Tag, Space, Typography, Select, message, Tooltip, Descriptions, Drawer } from 'antd'
import ReactECharts from 'echarts-for-react'
import { useSearchParams } from 'react-router-dom'
import { tracesApi } from '@/api/traces'
import type { TraceSummary, Span } from '@/api/traces'

const { Text } = Typography

interface SpanNode extends Span {
  depth: number
  children: SpanNode[]
  totalOffset: number
}

const SERVICE_COLORS = ['#1677ff', '#52c41a', '#faad14', '#ff4d4f', '#722ed1', '#13c2c2', '#eb2f96', '#fa8c16', '#2f54eb', '#a0d911']

function buildSpanTree(spans: Span[]): SpanNode[] {
  const map = new Map<string, SpanNode>()
  const roots: SpanNode[] = []

  const minTime = Math.min(...spans.map((s) => new Date(s.timestamp).getTime()))

  spans.forEach((s) => {
    map.set(s.span_id, {
      ...s,
      depth: 0,
      children: [],
      totalOffset: new Date(s.timestamp).getTime() - minTime,
    })
  })

  spans.forEach((s) => {
    const node = map.get(s.span_id)!
    if (s.parent_span_id && map.has(s.parent_span_id)) {
      map.get(s.parent_span_id)!.children.push(node)
    } else {
      roots.push(node)
    }
  })

  function setDepth(node: SpanNode, depth: number) {
    node.depth = depth
    node.children.sort((a, b) => a.totalOffset - b.totalOffset)
    node.children.forEach((c) => setDepth(c, depth + 1))
  }
  roots.forEach((r) => setDepth(r, 0))

  const flat: SpanNode[] = []
  function flatten(node: SpanNode) {
    flat.push(node)
    node.children.forEach(flatten)
  }
  roots.forEach(flatten)

  return flat
}

export default function Traces() {
  const [searchParams] = useSearchParams()
  const [traces, setTraces] = useState<TraceSummary[]>([])
  const [selectedTrace, setSelectedTrace] = useState<Span[]>([])
  const [selectedTraceID, setSelectedTraceID] = useState<string>('')
  const [services, setServices] = useState<string[]>([])
  const [serviceFilter, setServiceFilter] = useState<string>('')
  const [loading, setLoading] = useState(false)
  const [detailOpen, setDetailOpen] = useState(false)
  const [selectedSpan, setSelectedSpan] = useState<Span | null>(null)

  useEffect(() => {
    const controller = new AbortController()
    loadTraces()
    loadServices()
    return () => controller.abort()
  }, [serviceFilter])

  useEffect(() => {
    const traceId = searchParams.get('trace_id')
    if (traceId) {
      loadTraceDetail(traceId)
    }
  }, [searchParams])

  const loadTraces = async () => {
    setLoading(true)
    try {
      const res = await tracesApi.list({ limit: 100, service: serviceFilter || undefined })
      setTraces(res.data || [])
    } catch (e: unknown) {
      if ((e as Error)?.name === 'CanceledError') return
      message.error('加载链路列表失败')
    } finally {
      setLoading(false)
    }
  }

  const loadServices = async () => {
    try {
      const res = await tracesApi.services()
      setServices(res.data || [])
    } catch {
      // Services list is optional
    }
  }

  const loadTraceDetail = async (traceID: string) => {
    try {
      const res = await tracesApi.detail(traceID)
      setSelectedTrace(res.data || [])
      setSelectedTraceID(traceID)
    } catch (e: unknown) {
      if ((e as Error)?.name === 'CanceledError') return
      message.error('加载链路详情失败')
    }
  }

  const serviceColorMap = useMemo(() => {
    const allServices = [...new Set(selectedTrace.map((s) => s.service))]
    const map: Record<string, string> = {}
    allServices.forEach((s, i) => {
      map[s] = SERVICE_COLORS[i % SERVICE_COLORS.length]
    })
    return map
  }, [selectedTrace])

  const waterfallData = useMemo(() => {
    if (selectedTrace.length === 0) return []
    return buildSpanTree(selectedTrace)
  }, [selectedTrace])

  const getGanttOption = () => {
    if (waterfallData.length === 0) return null

    const maxDuration = Math.max(...waterfallData.map((s) => s.totalOffset + s.duration_ms))

    const data = waterfallData.map((span) => {
      const start = span.totalOffset
      const end = start + span.duration_ms
      const indent = '\u00A0\u00A0'.repeat(span.depth)
      return {
        name: `${indent}${span.service}: ${span.operation}`,
        value: [start, end],
        itemStyle: {
          color: serviceColorMap[span.service] || '#1677ff',
          borderRadius: 3,
        },
        span,
      }
    })

    return {
      tooltip: {
        formatter: (params: { data: { span: SpanNode } }) => {
          const { span } = params.data
          return `<div style="font-size:12px;max-width:400px">
            <b>${span.service}: ${span.operation}</b><br/>
            Span ID: ${span.span_id}<br/>
            Parent: ${span.parent_span_id || '(root)'}<br/>
            Duration: ${span.duration_ms.toFixed(2)}ms<br/>
            Offset: ${span.totalOffset.toFixed(0)}ms<br/>
            Status: ${span.status_code}
          </div>`
        },
      },
      grid: { left: '18%', right: '4%', top: '3%', bottom: '8%' },
      xAxis: {
        type: 'value',
        name: 'ms',
        min: 0,
        max: Math.ceil(maxDuration * 1.1),
        axisLabel: { formatter: (val: number) => val.toFixed(0) },
      },
      yAxis: {
        type: 'category',
        data: data.map((d) => d.name),
        axisLabel: {
          fontSize: 11,
          fontFamily: 'Menlo, Monaco, Consolas, monospace',
          width: 200,
          overflow: 'truncate',
        },
      },
      series: [
        {
          type: 'custom',
          renderItem: (_params: unknown, api: { coord: (v: number[]) => number[]; value: (i: number) => number; size: (v: number[][]) => number[]; style: () => Record<string, unknown> }) => {
            const start = api.coord([api.value(0), 0])
            const end = api.coord([api.value(1), 0])
            const height = api.size([[0, 1]])[1] * 0.6

            return {
              type: 'rect',
              shape: {
                x: start[0],
                y: start[1] - height / 2,
                width: Math.max(end[0] - start[0], 2),
                height: height,
              },
              style: api.style(),
            }
          },
          encode: { x: [0, 1] },
          data,
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
          <Tag style={{ cursor: 'pointer', fontFamily: 'monospace' }} onClick={() => loadTraceDetail(id)}>
            {id.substring(0, 16)}...
          </Tag>
        </Tooltip>
      ),
    },
    { title: '服务', dataIndex: 'root_service', key: 'root_service', width: 120 },
    { title: '操作', dataIndex: 'root_operation', key: 'root_operation', ellipsis: true },
    { title: 'Spans', dataIndex: 'span_count', key: 'span_count', width: 70 },
    {
      title: '耗时', dataIndex: 'duration_ms', key: 'duration_ms', width: 100,
      render: (ms: number) => (
        <Text code style={{ color: ms > 1000 ? '#ff4d4f' : ms > 500 ? '#faad14' : '#52c41a' }}>
          {ms.toFixed(1)}ms
        </Text>
      ),
    },
    {
      title: '状态', dataIndex: 'status_code', key: 'status_code', width: 80,
      render: (s: string) => <Tag color={s === 'ERROR' ? 'red' : 'green'}>{s}</Tag>,
    },
    {
      title: '时间', dataIndex: 'start_time', key: 'start_time', width: 170,
      render: (t: string) => new Date(t).toLocaleString('zh-CN'),
    },
  ]

  return (
    <div>
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

      {selectedTrace.length > 0 && (
        <Card
          title={<Space><span>链路详情: {selectedTraceID.substring(0, 16)}...</span><Tag>{selectedTrace.length} spans</Tag></Space>}
          size="small"
          extra={
            <Space>
              {Object.entries(serviceColorMap).map(([svc, color]) => (
                <Tag key={svc} color={color} style={{ borderRadius: 3 }}>{svc}</Tag>
              ))}
            </Space>
          }
        >
          <ReactECharts
            option={getGanttOption()}
            style={{ height: Math.max(waterfallData.length * 32 + 80, 200) }}
            onEvents={{
              click: (params: { data?: { span?: Span } }) => {
                if (params.data?.span) {
                  setSelectedSpan(params.data.span)
                  setDetailOpen(true)
                }
              },
            }}
          />
        </Card>
      )}

      <Drawer title="Span 详情" open={detailOpen} onClose={() => setDetailOpen(false)} width={480} destroyOnClose>
        {selectedSpan && (
          <Descriptions column={1} size="small" bordered>
            <Descriptions.Item label="Span ID"><Text code copyable>{selectedSpan.span_id}</Text></Descriptions.Item>
            <Descriptions.Item label="Parent Span ID"><Text code copyable>{selectedSpan.parent_span_id || '(root)'}</Text></Descriptions.Item>
            <Descriptions.Item label="Trace ID"><Text code copyable>{selectedSpan.trace_id}</Text></Descriptions.Item>
            <Descriptions.Item label="服务"><Tag color={serviceColorMap[selectedSpan.service]}>{selectedSpan.service}</Tag></Descriptions.Item>
            <Descriptions.Item label="操作">{selectedSpan.operation}</Descriptions.Item>
            <Descriptions.Item label="耗时"><Text strong style={{ color: selectedSpan.duration_ms > 500 ? '#ff4d4f' : '#52c41a' }}>{selectedSpan.duration_ms.toFixed(2)}ms</Text></Descriptions.Item>
            <Descriptions.Item label="状态"><Tag color={selectedSpan.status_code === 'ERROR' ? 'red' : 'green'}>{selectedSpan.status_code}</Tag></Descriptions.Item>
            <Descriptions.Item label="时间">{new Date(selectedSpan.timestamp).toLocaleString('zh-CN')}</Descriptions.Item>
            {selectedSpan.attributes && Object.keys(selectedSpan.attributes).length > 0 && (
              <Descriptions.Item label="属性">
                <pre style={{ margin: 0, fontSize: 11, maxHeight: 200, overflow: 'auto', background: '#fafafa', padding: 8, borderRadius: 4 }}>
                  {JSON.stringify(selectedSpan.attributes, null, 2)}
                </pre>
              </Descriptions.Item>
            )}
          </Descriptions>
        )}
      </Drawer>
    </div>
  )
}
