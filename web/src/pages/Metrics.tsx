import { useState, useRef } from 'react'
import { Card, Input, Button, Space, Table, Tag, Typography, Select, Tooltip, message, Row, Col } from 'antd'
import { SearchOutlined, ReloadOutlined, LineChartOutlined } from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import { metricsApi } from '@/api'
import type { MetricSeries } from '@/api/metrics'

const { Title, Text } = Typography
const { TextArea } = Input

// Common PromQL queries for quick access
const commonQueries = [
  { name: '系统存活', query: 'up', description: '检查所有目标是否存活' },
  { name: 'CPU 使用率', query: '100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)', description: '主机 CPU 使用率' },
  { name: '内存使用率', query: '(1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100', description: '主机内存使用率' },
  { name: '磁盘使用率', query: '(1 - node_filesystem_avail_bytes{mountpoint="/"} / node_filesystem_size_bytes{mountpoint="/"}) * 100', description: '根分区磁盘使用率' },
  { name: '网络接收', query: 'rate(node_network_receive_bytes_total[5m])', description: '网络接收速率' },
  { name: '磁盘 I/O', query: 'rate(node_disk_io_time_seconds_total[5m])', description: '磁盘 I/O 使用率' },
]

// Color palette for multiple series
const COLORS = ['#1677ff', '#ff4d4f', '#52c41a', '#faad14', '#722ed1', '#13c2c2', '#eb2f96', '#fa8c16']

export default function Metrics() {
  const [query, setQuery] = useState('up')
  const [results, setResults] = useState<MetricSeries[]>([])
  const [loading, setLoading] = useState(false)
  const [showChart, setShowChart] = useState(true)
  const [timeRange, setTimeRange] = useState('1h')
  const [step, setStep] = useState('60s')
  const chartRef = useRef<any>(null)

  const handleQuery = async () => {
    if (!query.trim()) return
    setLoading(true)
    try {
      const now = Math.floor(Date.now() / 1000)
      const rangeSeconds: Record<string, number> = {
        '15m': 900, '30m': 1800, '1h': 3600, '3h': 10800, '6h': 21600, '12h': 43200, '24h': 86400, '7d': 604800,
      }
      const start = now - (rangeSeconds[timeRange] || 3600)

      const res = await metricsApi.query({
        query,
        start: String(start),
        end: String(now),
        step,
      })
      setResults(res.data || [])
      if (res.data?.length === 0) {
        message.info('查询无结果')
      }
    } catch (e: any) {
      message.error(e?.error || '查询失败')
    } finally {
      setLoading(false)
    }
  }

  // Build ECharts option from query results
  const getChartOption = () => {
    if (results.length === 0) return null

    const series = results.map((r, i) => {
      const label = Object.entries(r.metric)
        .filter(([k]) => k !== '__name__')
        .map(([k, v]) => `${k}="${v}"`)
        .join(', ')

      return {
        name: label || `series_${i}`,
        type: 'line',
        smooth: true,
        symbol: 'none',
        lineStyle: { width: 1.5 },
        itemStyle: { color: COLORS[i % COLORS.length] },
        data: r.values.map((v) => [v.timestamp * 1000, v.value]),
      }
    })

    return {
      tooltip: {
        trigger: 'axis',
        formatter: (params: any[]) => {
          if (!params.length) return ''
          const time = new Date(params[0].data[0]).toLocaleString('zh-CN')
          let html = `<div style="font-size:12px"><b>${time}</b></div>`
          params.forEach((p: any) => {
            html += `<div style="display:flex;align-items:center;gap:4px">`
            html += `<span style="display:inline-block;width:8px;height:8px;border-radius:50%;background:${p.color}"></span>`
            html += `<span>${p.seriesName}: <b>${p.data[1]?.toFixed(4)}</b></span></div>`
          })
          return html
        },
      },
      legend: {
        type: 'scroll',
        bottom: 0,
        textStyle: { fontSize: 11 },
        pageTextStyle: { color: '#666' },
      },
      grid: { left: '3%', right: '4%', bottom: '12%', top: '5%', containLabel: true },
      xAxis: {
        type: 'time',
        axisLabel: {
          fontSize: 10,
          formatter: (val: number) => {
            const d = new Date(val)
            return `${d.getHours()}:${String(d.getMinutes()).padStart(2, '0')}`
          },
        },
      },
      yAxis: {
        type: 'value',
        axisLabel: { fontSize: 10 },
        splitLine: { lineStyle: { type: 'dashed' } },
      },
      dataZoom: [
        { type: 'inside', start: 0, end: 100 },
        { type: 'slider', start: 0, end: 100, height: 20, bottom: 30 },
      ],
      series,
    }
  }

  const columns = [
    {
      title: '指标标签',
      dataIndex: 'metric',
      key: 'metric',
      ellipsis: true,
      render: (m: Record<string, string>) =>
        Object.entries(m)
          .filter(([k]) => k !== '__name__')
          .map(([k, v]) => (
            <Tag key={k}>
              {k}={v}
            </Tag>
          )) || '-',
    },
    {
      title: '名称',
      dataIndex: 'metric',
      key: 'name',
      width: 200,
      render: (m: Record<string, string>) => <Text strong>{m.__name__ || '-'}</Text>,
    },
    {
      title: '最新值',
      key: 'value',
      width: 120,
      render: (_: any, record: MetricSeries) => {
        const last = record.values?.[record.values.length - 1]
        return last ? <Text code>{last.value.toFixed(4)}</Text> : '-'
      },
    },
    {
      title: '数据点',
      key: 'count',
      width: 80,
      render: (_: any, record: MetricSeries) => record.values?.length || 0,
    },
  ]

  return (
    <div>
      <Title level={4}>
        <LineChartOutlined /> 指标监控
      </Title>

      {/* Query editor */}
      <Card size="small" style={{ marginBottom: 16 }}>
        <Row gutter={16} align="middle">
          <Col flex="auto">
            <TextArea
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="输入 PromQL 查询..."
              autoSize={{ minRows: 1, maxRows: 4 }}
              style={{ fontFamily: 'monospace', fontSize: 13 }}
              onPressEnter={(e) => {
                if (!e.shiftKey) {
                  e.preventDefault()
                  handleQuery()
                }
              }}
            />
          </Col>
          <Col>
            <Space direction="vertical" size={4}>
              <Space size={4}>
                <Select
                  value={timeRange}
                  onChange={setTimeRange}
                  style={{ width: 90 }}
                  size="small"
                  options={[
                    { label: '15 分钟', value: '15m' },
                    { label: '30 分钟', value: '30m' },
                    { label: '1 小时', value: '1h' },
                    { label: '3 小时', value: '3h' },
                    { label: '6 小时', value: '6h' },
                    { label: '12 小时', value: '12h' },
                    { label: '24 小时', value: '24h' },
                    { label: '7 天', value: '7d' },
                  ]}
                />
                <Select
                  value={step}
                  onChange={setStep}
                  style={{ width: 80 }}
                  size="small"
                  options={[
                    { label: '15s', value: '15s' },
                    { label: '1m', value: '60s' },
                    { label: '5m', value: '300s' },
                    { label: '15m', value: '900s' },
                    { label: '1h', value: '3600s' },
                  ]}
                />
              </Space>
              <Space size={4}>
                <Button type="primary" icon={<SearchOutlined />} onClick={handleQuery} loading={loading} size="small">
                  查询
                </Button>
                <Button icon={<ReloadOutlined />} onClick={() => setResults([])} size="small">
                  清空
                </Button>
                <Tooltip title={showChart ? '隐藏图表' : '显示图表'}>
                  <Button
                    type={showChart ? 'primary' : 'default'}
                    icon={<LineChartOutlined />}
                    onClick={() => setShowChart(!showChart)}
                    size="small"
                  />
                </Tooltip>
              </Space>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* Quick queries */}
      <Card size="small" title="常用查询" style={{ marginBottom: 16 }} bodyStyle={{ padding: '8px 16px' }}>
        <Space wrap size={[8, 8]}>
          {commonQueries.map((q) => (
            <Tooltip key={q.name} title={q.description}>
              <Tag
                style={{ cursor: 'pointer' }}
                onClick={() => {
                  setQuery(q.query)
                }}
              >
                {q.name}
              </Tag>
            </Tooltip>
          ))}
        </Space>
      </Card>

      {/* Chart */}
      {showChart && results.length > 0 && (
        <Card
          size="small"
          title={`查询结果 - ${results.length} 条序列`}
          style={{ marginBottom: 16 }}
          extra={
            <Space>
              <Tag>{timeRange}</Tag>
              <Tag>step: {step}</Tag>
            </Space>
          }
        >
          <ReactECharts
            ref={chartRef}
            option={getChartOption()}
            style={{ height: 400 }}
            notMerge
            lazyUpdate
          />
        </Card>
      )}

      {/* Results table */}
      <Card title={`序列列表 (${results.length})`} size="small">
        <Table
          dataSource={results}
          columns={columns}
          rowKey={(_, i) => String(i)}
          size="small"
          pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 条` }}
          expandable={{
            expandedRowRender: (record) => (
              <div style={{ maxHeight: 200, overflow: 'auto' }}>
                <table style={{ width: '100%', fontSize: 12 }}>
                  <thead>
                    <tr>
                      <th>时间</th>
                      <th>值</th>
                    </tr>
                  </thead>
                  <tbody>
                    {record.values?.slice(-20).map((v, i) => (
                      <tr key={i}>
                        <td>{new Date(v.timestamp * 1000).toLocaleString('zh-CN')}</td>
                        <td><Text code>{v.value.toFixed(4)}</Text></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ),
          }}
        />
      </Card>
    </div>
  )
}
