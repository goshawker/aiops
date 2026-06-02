import { useState, useEffect } from 'react'
import { Card, Typography, Tag, Space, Spin } from 'antd'
import { ClusterOutlined } from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import client from '@/api/client'

const { Text } = Typography

interface GraphData {
  nodes: string[]
  edges: Array<{
    source: string
    target: string
    confidence: number
    lag: number
  }>
  metric_count: number
  last_discovery: number
}

const NODE_COLORS: Record<string, string> = {
  host: '#52c41a',
  service: '#1677ff',
  metric: '#faad14',
  component: '#722ed1',
  alert: '#ff4d4f',
}

export default function Topology() {
  const [graph, setGraph] = useState<GraphData | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadGraph()
  }, [])

  const loadGraph = async () => {
    setLoading(true)
    try {
      const res = await client.get('/rca/graph')
      setGraph(res.data)
    } catch (e) {
      // Service may not be running
    } finally {
      setLoading(false)
    }
  }

  const getOption = () => {
    if (!graph || graph.nodes.length === 0) return null

    const nodes = graph.nodes.map((name) => {
      const type = name.startsWith('host-') ? 'host'
        : name.startsWith('svc-') ? 'service'
        : name.startsWith('cmp-') ? 'component'
        : 'metric'

      return {
        id: name,
        name: name.replace(/^(host-|svc-|cmp-)/, ''),
        category: type,
        symbolSize: type === 'service' ? 40 : type === 'host' ? 35 : 25,
        itemStyle: { color: NODE_COLORS[type] || '#999' },
      }
    })

    const categories = [
      { name: '主机', itemStyle: { color: NODE_COLORS.host } },
      { name: '服务', itemStyle: { color: NODE_COLORS.service } },
      { name: '组件', itemStyle: { color: NODE_COLORS.component } },
      { name: '指标', itemStyle: { color: NODE_COLORS.metric } },
    ]

    const links = graph.edges.map((e) => ({
      source: e.source,
      target: e.target,
      value: e.confidence,
      lineStyle: {
        width: Math.max(e.confidence * 2, 1),
        curveness: 0.1,
      },
    }))

    return {
      tooltip: {
        trigger: 'item',
        formatter: (params: any) => {
          if (params.dataType === 'edge') {
            return `${params.data.source} → ${params.data.target}<br/>置信度: ${((params.data.value || 0) * 100).toFixed(0)}%`
          }
          return `${params.name}<br/>类型: ${params.data.category}`
        },
      },
      legend: {
        data: categories.map((c) => c.name),
        bottom: 0,
      },
      series: [
        {
          type: 'graph',
          layout: 'force',
          data: nodes,
          links: links,
          categories: categories,
          roam: true,
          draggable: true,
          label: {
            show: true,
            fontSize: 11,
            position: 'bottom',
          },
          force: {
            repulsion: 300,
            edgeLength: [80, 200],
            gravity: 0.1,
          },
          lineStyle: {
            color: '#999',
            opacity: 0.6,
          },
          emphasis: {
            focus: 'adjacency',
            lineStyle: { width: 4, opacity: 1 },
          },
        },
      ],
    }
  }

  return (
    <div>
      {graph && (
        <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 16 }}>
          <Space>
            <Tag>{graph.nodes.length} 节点</Tag>
            <Tag>{graph.edges.length} 关系</Tag>
            <Tag>{graph.metric_count} 指标</Tag>
          </Space>
        </div>
      )}

      <Card style={{ padding: 16 }}>
        {loading ? (
          <div style={{ height: 500, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Spin size="large" />
          </div>
        ) : graph && graph.nodes.length > 0 ? (
          <ReactECharts option={getOption()} style={{ height: 500 }} />
        ) : (
          <div style={{ height: 500, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', color: '#999' }}>
            <ClusterOutlined style={{ fontSize: 48, marginBottom: 16 }} />
            <Text type="secondary">暂无拓扑数据</Text>
            <Text type="secondary" style={{ fontSize: 12 }}>请先摄入指标数据并执行因果发现</Text>
          </div>
        )}
      </Card>
    </div>
  )
}
