import { useState, useRef } from 'react'
import { Card, Typography, Input, Button, Space, Tag, message, Row, Col, List, Progress, Tooltip } from 'antd'
import { SearchOutlined, AimOutlined } from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import client from '@/api/client'

const { Text } = Typography
const { TextArea } = Input

interface RootCause {
  metric_name: string
  score: number
  reason: string
  related_metrics: string[]
  evidence: string[]
}

interface CausalGraph {
  nodes: string[]
  edges: Array<{
    source: string
    target: string
    confidence: number
    lag: number
  }>
}

export default function RCA() {
  const [affectedMetrics, setAffectedMetrics] = useState('')
  const [results, setResults] = useState<RootCause[]>([])
  const [graph, setGraph] = useState<CausalGraph | null>(null)
  const [loading, setLoading] = useState(false)
  const [highlightNode, setHighlightNode] = useState<string | null>(null)
  const chartRef = useRef<any>(null)

  const handleAnalyze = async () => {
    const metrics = affectedMetrics
      .split('\n')
      .map((s) => s.trim())
      .filter(Boolean)

    if (metrics.length === 0) {
      message.warning('请输入受影响的指标')
      return
    }

    setLoading(true)
    try {
      const res = await client.post('/rca/analyze', { affected_metrics: metrics })
      setResults(res.data?.root_causes || [])

      // Also fetch graph
      try {
        const graphRes = await client.get('/rca/graph')
        setGraph(graphRes.data)
      } catch (e) {
        // Graph may not be available
      }
    } catch (e) {
      message.error('分析失败')
    } finally {
      setLoading(false)
    }
  }

  const getGraphOption = () => {
    if (!graph || graph.nodes.length === 0) return null

    const affectedList = affectedMetrics.split('\n').map((s) => s.trim()).filter(Boolean)

    const categories = [{ name: '指标' }]
    const nodes = graph.nodes.map((name) => {
      const isHighlighted = highlightNode === name
      const isAffected = affectedList.some((m) => name.includes(m))
      const isRootCause = results.some((r) => name.includes(r.metric_name))

      let color = '#1677ff'
      let size = 30
      if (isRootCause) { color = '#ff4d4f'; size = 40 }
      else if (isAffected) { color = '#faad14'; size = 35 }
      if (isHighlighted) { size = 50 }

      return {
        id: name,
        name: name.split('{')[0],
        category: 0,
        symbolSize: size,
        itemStyle: {
          color,
          borderColor: isHighlighted ? '#000' : undefined,
          borderWidth: isHighlighted ? 3 : 0,
          shadowBlur: isHighlighted ? 20 : 0,
          shadowColor: isHighlighted ? color : undefined,
        },
      }
    })

    const links = graph.edges.map((e) => ({
      source: e.source,
      target: e.target,
      value: e.confidence,
      lineStyle: {
        width: e.confidence * 3,
        curveness: 0.2,
      },
    }))

    return {
      tooltip: {
        trigger: 'item',
        formatter: (params: any) => {
          if (params.dataType === 'edge') {
            return `${params.data.source} → ${params.data.target}<br/>置信度: ${(params.data.value * 100).toFixed(0)}%`
          }
          return params.name
        },
      },
      series: [
        {
          type: 'graph',
          layout: 'force',
          data: nodes,
          links: links,
          categories: categories,
          roam: true,
          label: { show: true, fontSize: 11 },
          force: { repulsion: 200, edgeLength: 120 },
          lineStyle: { color: '#1677ff', opacity: 0.6 },
          emphasis: {
            focus: 'adjacency',
            lineStyle: { width: 4 },
          },
        },
      ],
    }
  }

  const handleHighlightRootCause = (metricName: string) => {
    setHighlightNode(metricName)
    // Trigger chart update via ref
    if (chartRef.current) {
      const instance = chartRef.current.getEchartsInstance()
      if (instance) {
        instance.dispatchAction({ type: 'downplay' })
        // Find the node that matches this metric
        const matchingNode = graph?.nodes.find((n) => n.includes(metricName))
        if (matchingNode) {
          instance.dispatchAction({ type: 'highlight', name: matchingNode.split('{')[0] })
        }
      }
    }
  }

  return (
    <div>
      {/* Input */}
      <Card size="small" style={{ marginBottom: 16, padding: 16 }}>
        <Row gutter={16}>
          <Col flex="auto">
            <TextArea
              value={affectedMetrics}
              onChange={(e) => setAffectedMetrics(e.target.value)}
              placeholder="输入受影响的指标（每行一个）&#10;例如：&#10;cpu_usage&#10;memory_usage&#10;error_rate"
              autoSize={{ minRows: 3, maxRows: 6 }}
              style={{ fontFamily: 'monospace', fontSize: 13 }}
            />
          </Col>
          <Col>
            <Button
              type="primary"
              icon={<SearchOutlined />}
              onClick={handleAnalyze}
              loading={loading}
            >
              分析根因
            </Button>
          </Col>
        </Row>
      </Card>

      {/* Results */}
      {results.length > 0 && (
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col xs={24} lg={12}>
            <Card title="根因排名" size="small">
              <List
                dataSource={results}
                renderItem={(item: RootCause, index: number) => (
                  <List.Item
                    style={{
                      cursor: 'pointer', padding: '12px 8px', borderRadius: 6, transition: 'background 0.2s',
                      background: highlightNode === item.metric_name ? '#f0f5ff' : undefined,
                    }}
                    onClick={() => handleHighlightRootCause(item.metric_name)}
                  >
                    <div style={{ width: '100%' }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                        <Space>
                          <Tag color={index === 0 ? 'red' : index === 1 ? 'orange' : 'blue'}>
                            #{index + 1}
                          </Tag>
                          <Text strong>{item.metric_name}</Text>
                          <Tooltip title="在因果图中高亮">
                            <AimOutlined style={{ color: '#1677ff', fontSize: 12 }} />
                          </Tooltip>
                        </Space>
                        <Text type="secondary">{(item.score * 100).toFixed(0)}%</Text>
                      </div>
                      <Progress
                        percent={Math.round(item.score * 100)}
                        strokeColor={index === 0 ? '#ff4d4f' : index === 1 ? '#faad14' : '#1677ff'}
                        showInfo={false}
                        size="small"
                      />
                      <Text type="secondary" style={{ fontSize: 12 }}>{item.reason}</Text>
                      {item.evidence.length > 0 && (
                        <div style={{ marginTop: 4 }}>
                          {item.evidence.map((e, i) => (
                            <Tag key={i} style={{ fontSize: 11 }}>{e}</Tag>
                          ))}
                        </div>
                      )}
                      {item.related_metrics.length > 0 && (
                        <div style={{ marginTop: 4 }}>
                          <Text type="secondary" style={{ fontSize: 11 }}>关联指标: </Text>
                          {item.related_metrics.map((m, i) => (
                            <Tag key={i} color="blue" style={{ fontSize: 11 }}>{m}</Tag>
                          ))}
                        </div>
                      )}
                    </div>
                  </List.Item>
                )}
              />
            </Card>
          </Col>

          {/* Causal graph */}
          <Col xs={24} lg={12}>
            <Card
              title="因果关系图"
              size="small"
              extra={
                <Space size={4}>
                  <Tag color="red">根因</Tag>
                  <Tag color="orange">受影响</Tag>
                  <Tag color="blue">其他</Tag>
                </Space>
              }
            >
              {graph && graph.nodes.length > 0 ? (
                <ReactECharts ref={chartRef} option={getGraphOption()} style={{ height: 350 }} />
              ) : (
                <div style={{ height: 350, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#999' }}>
                  暂无因果图数据，请先摄入指标数据
                </div>
              )}
            </Card>
          </Col>
        </Row>
      )}

      {/* Graph only view */}
      {results.length === 0 && graph && graph.nodes.length > 0 && (
        <Card title="因果关系图" size="small">
          <ReactECharts ref={chartRef} option={getGraphOption()} style={{ height: 400 }} />
        </Card>
      )}
    </div>
  )
}
