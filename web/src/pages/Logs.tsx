import { useState, useEffect } from 'react'
import { Card, Input, Select, Table, Tag, Space, Typography, message, Tooltip } from 'antd'
import { SearchOutlined, FileTextOutlined } from '@ant-design/icons'
import { logsApi } from '@/api'
import type { LogEntry } from '@/api/logs'

const { Title, Text, Paragraph } = Typography

const levelColors: Record<string, string> = {
  FATAL: 'red',
  ERROR: 'red',
  WARN: 'orange',
  WARNING: 'orange',
  INFO: 'blue',
  DEBUG: 'default',
}

// Common services for quick filter
const commonServices = [
  { label: 'gateway', value: 'gateway' },
  { label: 'query', value: 'query' },
  { label: 'alert', value: 'alert' },
  { label: 'anomaly', value: 'anomaly' },
  { label: 'alert-agg', value: 'alert-agg' },
  { label: 'llm', value: 'llm' },
  { label: 'node-exporter', value: 'node-exporter' },
]

export default function Logs() {
  const [query, setQuery] = useState('')
  const [level, setLevel] = useState<string>('')
  const [service, setService] = useState<string>('')
  const [results, setResults] = useState<LogEntry[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState({ limit: 50, offset: 0 })

  useEffect(() => {
    handleSearch()
  }, [])

  const handleSearch = async (offset = 0) => {
    setLoading(true)
    try {
      const res = await logsApi.search({
        q: query || undefined,
        level: level || undefined,
        service: service || undefined,
        limit: page.limit,
        offset,
      })
      setResults(res.data || [])
      setTotal(res.total || 0)
      setPage((p) => ({ ...p, offset }))
    } catch (e: any) {
      message.error(e?.error || '日志查询失败')
    } finally {
      setLoading(false)
    }
  }

  const columns = [
    {
      title: '时间',
      dataIndex: 'timestamp',
      key: 'timestamp',
      width: 180,
      render: (t: string) => new Date(t).toLocaleString('zh-CN'),
    },
    {
      title: '级别',
      dataIndex: 'level',
      key: 'level',
      width: 80,
      render: (l: string) => <Tag color={levelColors[l] || 'default'}>{l}</Tag>,
    },
    {
      title: '服务',
      dataIndex: 'service',
      key: 'service',
      width: 120,
      render: (s: string) => s || '-',
    },
    {
      title: '主机',
      dataIndex: 'host',
      key: 'host',
      width: 120,
      render: (h: string) => h || '-',
    },
    {
      title: '日志内容',
      dataIndex: 'message',
      key: 'message',
      ellipsis: true,
    },
    {
      title: 'TraceID',
      dataIndex: 'trace_id',
      key: 'trace_id',
      width: 160,
      render: (t: string) =>
        t ? (
          <Tooltip title={t}>
            <Tag>{t.substring(0, 16)}...</Tag>
          </Tooltip>
        ) : (
          '-'
        ),
    },
  ]

  return (
    <div>
      <Title level={4}>
        <FileTextOutlined /> 日志管理
      </Title>

      {/* Search filters */}
      <Card size="small" style={{ marginBottom: 16 }}>
        <Space wrap>
          <Input
            placeholder="搜索关键词"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onPressEnter={() => handleSearch()}
            prefix={<SearchOutlined />}
            style={{ width: 300 }}
            allowClear
          />
          <Select
            placeholder="日志级别"
            value={level || undefined}
            onChange={(v) => setLevel(v || '')}
            allowClear
            style={{ width: 120 }}
            options={[
              { label: 'FATAL', value: 'FATAL' },
              { label: 'ERROR', value: 'ERROR' },
              { label: 'WARN', value: 'WARN' },
              { label: 'INFO', value: 'INFO' },
              { label: 'DEBUG', value: 'DEBUG' },
            ]}
          />
          <Select
            placeholder="服务名称"
            value={service || undefined}
            onChange={(v) => setService(v || '')}
            allowClear
            showSearch
            style={{ width: 150 }}
            options={commonServices}
          />
          <Select
            value={page.limit}
            onChange={(v) => setPage((p) => ({ ...p, limit: v }))}
            style={{ width: 100 }}
            options={[
              { label: '50 条', value: 50 },
              { label: '100 条', value: 100 },
              { label: '200 条', value: 200 },
            ]}
          />
          <Space>
            <SearchOutlined
              style={{ cursor: 'pointer', fontSize: 16, color: '#1677ff' }}
              onClick={() => handleSearch()}
            />
          </Space>
        </Space>
      </Card>

      {/* Results table */}
      <Card title={`日志列表 (${total} 条)`} size="small">
        <Table
          dataSource={results}
          columns={columns}
          rowKey={(r) => `${r.timestamp}-${r.host}-${r.message?.substring(0, 20)}`}
          loading={loading}
          size="small"
          scroll={{ x: 900 }}
          pagination={{
            total,
            current: Math.floor(page.offset / page.limit) + 1,
            pageSize: page.limit,
            showSizeChanger: false,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (p) => handleSearch((p - 1) * page.limit),
          }}
          expandable={{
            expandedRowRender: (record) => (
              <div style={{ padding: '8px 0' }}>
                <Space direction="vertical" size={4} style={{ width: '100%' }}>
                  <div>
                    <Text strong>完整日志：</Text>
                    <Paragraph
                      style={{
                        marginTop: 4,
                        padding: 8,
                        background: '#f5f5f5',
                        borderRadius: 4,
                        fontFamily: 'monospace',
                        fontSize: 12,
                        whiteSpace: 'pre-wrap',
                        wordBreak: 'break-all',
                      }}
                    >
                      {record.message}
                    </Paragraph>
                  </div>
                  <Space wrap size={[16, 4]}>
                    <Text type="secondary">服务: {record.service || '-'}</Text>
                    <Text type="secondary">主机: {record.host || '-'}</Text>
                    {record.trace_id && <Text type="secondary">TraceID: {record.trace_id}</Text>}
                    {record.span_id && <Text type="secondary">SpanID: {record.span_id}</Text>}
                  </Space>
                </Space>
              </div>
            ),
          }}
        />
      </Card>
    </div>
  )
}
