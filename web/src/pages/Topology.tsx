import { useState, useEffect, useMemo } from 'react'
import { Card, Typography, Tag, Space, Spin, Descriptions, Drawer, message } from 'antd'
import { ClusterOutlined } from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import { topologyApi } from '@/api/topology'
import type { GraphData } from '@/api/topology'

const { Text } = Typography

const NODE_COLORS: Record<string, string> = {
  host: '#52c41a',
  service: '#1677ff',
  metric: '#faad14',
  component: '#722ed1',
  alert: '#ff4d4f',
}

function getNodeType(name: string): string {
  if (name.startsWith('host-')) return 'host'
  if (name.startsWith('svc-')) return 'service'
  if (name.startsWith('cmp-')) return 'component'
  return 'metric'
}

const typeLabel: Record<string, string> = {
  host: '主机',
  service: '服务',
  component: '组件',
  metric: '指标',
}

export default function Topology() {
  const [graph, setGraph] = useState<GraphData | null>(null)
  const [loading, setLoading] = useState(true)
  const [detailOpen, setDetailOpen] = useState(false)
  const [selectedNode, setSelectedNode] = useState<string | null>(null)

  useEffect(() => {
    const controller = new AbortController()
    loadGraph()
    return () => controller.abort()
  }, [])

  const loadGraph = async () => {
    setLoading(true)
    try {
      const res = await topologyApi.graph()
      setGraph(res)
    } catch (e: any) {
      if (e?.name === 'CanceledError') return
      message.error('加载拓扑数据失败')
    } finally {
      setLoading(false)
    }
  }

  // Compute edges connected to selected node
  const nodeEdges = useMemo(() => {
    if (!selectedNode || !graph) return { incoming: [], outgoing: [] }
    const incoming = graph.edges.filter((e) => e.target === selectedNode)
    const outgoing = graph.edges.filter((e) => e.source === selectedNode)
    return { incoming, outgoing }
  }, [selectedNode, graph])

  const getOption = () => {
    if (!graph || graph.nodes.length === 0) return null

    const nodes = graph.nodes.map((name) => {
      const type = getNodeType(name)
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
        formatter: (params: { dataType?: string; data?: { source?: string; target?: string; value?: number; id?: string; name?: string } }) => {
          if (params.dataType === 'edge' && params.data) {
            return `${params.data.source} → ${params.data.target}<br/>置信度: ${((params.data.value || 0) * 100).toFixed(0)}%`
          }
          const type = getNodeType(params.data?.id || '')
          return `<b>${params.data?.name || ''}</b><br/>类型: ${typeLabel[type] || type}`
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
          <ReactECharts
            option={getOption()}
            style={{ height: 500 }}
            onEvents={{
              click: (params: { dataType?: string; data?: { id?: string } }) => {
                if (params.dataType === 'node' && params.data?.id) {
                  setSelectedNode(params.data.id)
                  setDetailOpen(true)
                }
              },
            }}
          />
        ) : (
          <div style={{ height: 500, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', color: '#999' }}>
            <ClusterOutlined style={{ fontSize: 48, marginBottom: 16 }} />
            <Text type="secondary">暂无拓扑数据</Text>
            <Text type="secondary" style={{ fontSize: 12 }}>请先摄入指标数据并执行因果发现</Text>
          </div>
        )}
      </Card>

      {/* Node detail drawer */}
      <Drawer
        title="节点详情"
        open={detailOpen}
        onClose={() => { setDetailOpen(false); setSelectedNode(null) }}
        width={480}
        destroyOnClose
      >
        {selectedNode && (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions column={1} size="small" bordered>
              <Descriptions.Item label="节点名称">
                <Text code copyable>{selectedNode}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="类型">
                <Tag color={NODE_COLORS[getNodeType(selectedNode)]}>
                  {typeLabel[getNodeType(selectedNode)]}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="关系数">
                {nodeEdges.incoming.length + nodeEdges.outgoing.length}
              </Descriptions.Item>
            </Descriptions>

            {nodeEdges.incoming.length > 0 && (
              <Card title={`上游节点 (${nodeEdges.incoming.length})`} size="small">
                {nodeEdges.incoming.map((e, i) => (
                  <div key={i} style={{ display: 'flex', justifyContent: 'space-between', padding: '4px 0', borderBottom: '1px solid #f0f0f0' }}>
                    <Space>
                      <Tag color={NODE_COLORS[getNodeType(e.source)]} style={{ fontSize: 11 }}>
                        {typeLabel[getNodeType(e.source)]}
                      </Tag>
                      <Text style={{ fontSize: 12 }}>{e.source.replace(/^(host-|svc-|cmp-)/, '')}</Text>
                    </Space>
                    <Tag>{(e.confidence * 100).toFixed(0)}%</Tag>
                  </div>
                ))}
              </Card>
            )}

            {nodeEdges.outgoing.length > 0 && (
              <Card title={`下游节点 (${nodeEdges.outgoing.length})`} size="small">
                {nodeEdges.outgoing.map((e, i) => (
                  <div key={i} style={{ display: 'flex', justifyContent: 'space-between', padding: '4px 0', borderBottom: '1px solid #f0f0f0' }}>
                    <Space>
                      <Tag color={NODE_COLORS[getNodeType(e.target)]} style={{ fontSize: 11 }}>
                        {typeLabel[getNodeType(e.target)]}
                      </Tag>
                      <Text style={{ fontSize: 12 }}>{e.target.replace(/^(host-|svc-|cmp-)/, '')}</Text>
                    </Space>
                    <Tag>{(e.confidence * 100).toFixed(0)}%</Tag>
                  </div>
                ))}
              </Card>
            )}
          </Space>
        )}
      </Drawer>
    </div>
  )
}
