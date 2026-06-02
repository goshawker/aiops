import { useEffect, useState } from 'react'
import {
  Card, Table, Tag, Button, Space, Tabs, message, Select, Row, Col, Statistic,
  Modal, Form, Input, Switch, Popconfirm,
} from 'antd'
import {
  CheckOutlined,
  CheckCircleOutlined,
  AlertOutlined,
  CloseCircleOutlined,
  WarningOutlined,
  ReloadOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SettingOutlined,
} from '@ant-design/icons'
import { alertsApi } from '@/api'
import type { AlertEvent, Incident, AlertRule } from '@/api/alerts'

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

const ruleTypeLabel: Record<string, string> = {
  threshold: '阈值',
  anomaly: '异常检测',
  log_pattern: '日志模式',
}

export default function Alerts() {
  const [events, setEvents] = useState<AlertEvent[]>([])
  const [incidents, setIncidents] = useState<Incident[]>([])
  const [rules, setRules] = useState<AlertRule[]>([])
  const [loading, setLoading] = useState(false)
  const [tab, setTab] = useState('incidents')
  const [severityFilter, setSeverityFilter] = useState<string>('')
  const [statusFilter, setStatusFilter] = useState<string>('')

  // Rule modal
  const [ruleModalOpen, setRuleModalOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<AlertRule | null>(null)
  const [ruleSaving, setRuleSaving] = useState(false)
  const [ruleForm] = Form.useForm()

  useEffect(() => {
    loadData()
  }, [tab])

  const loadData = async () => {
    setLoading(true)
    try {
      if (tab === 'incidents') {
        const res = await alertsApi.listIncidents({ limit: 200 })
        setIncidents(res.data || [])
      } else if (tab === 'events') {
        const res = await alertsApi.listEvents({ limit: 200 })
        setEvents(res.data || [])
      } else if (tab === 'rules') {
        const res = await alertsApi.listRules({ limit: 200 })
        setRules(res.data || [])
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

  // Rule CRUD
  const openCreateRule = () => {
    setEditingRule(null)
    ruleForm.resetFields()
    ruleForm.setFieldsValue({ severity: 'warning', enabled: true, rule_type: 'threshold' })
    setRuleModalOpen(true)
  }

  const openEditRule = (rule: AlertRule) => {
    setEditingRule(rule)
    ruleForm.setFieldsValue({
      name: rule.name,
      description: rule.description,
      rule_type: rule.rule_type,
      rule_config: rule.rule_config,
      severity: rule.severity,
      enabled: rule.enabled,
      notify_config: rule.notify_config,
      silence_config: rule.silence_config,
    })
    setRuleModalOpen(true)
  }

  const handleSaveRule = async () => {
    try {
      const values = await ruleForm.validateFields()
      setRuleSaving(true)
      if (editingRule) {
        await alertsApi.updateRule(editingRule.id, values)
        message.success('规则已更新')
      } else {
        await alertsApi.createRule(values)
        message.success('规则已创建')
      }
      setRuleModalOpen(false)
      loadData()
    } catch (e: any) {
      if (e?.errorFields) return
      message.error(e?.error || '保存失败')
    } finally {
      setRuleSaving(false)
    }
  }

  const handleDeleteRule = async (id: number) => {
    try {
      await alertsApi.deleteRule(id)
      message.success('规则已删除')
      loadData()
    } catch (e) {
      message.error('删除失败')
    }
  }

  const handleToggleEnabled = async (rule: AlertRule, checked: boolean) => {
    try {
      await alertsApi.updateRule(rule.id, { enabled: checked })
      setRules((prev) => prev.map((r) => (r.id === rule.id ? { ...r, enabled: checked } : r)))
      message.success(checked ? '规则已启用' : '规则已禁用')
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
  const enabledRuleCount = rules.filter((r) => r.enabled).length

  const incidentColumns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 70 },
    {
      title: '严重程度', dataIndex: 'severity', key: 'severity', width: 100,
      render: (s: string) => <Tag color={severityColor(s)}>{s}</Tag>,
    },
    { title: '标题', dataIndex: 'title', key: 'title', ellipsis: true },
    {
      title: '影响服务', dataIndex: 'affected_services', key: 'affected_services', width: 150,
      render: (s: string[]) => s?.length ? s.map((svc) => <Tag key={svc}>{svc}</Tag>) : '-',
    },
    { title: '事件数', dataIndex: 'event_count', key: 'event_count', width: 70 },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 110,
      render: (s: string) => <Tag color={statusColor(s)}>{s}</Tag>,
    },
    { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
    {
      title: '操作', key: 'action', width: 140,
      render: (_: any, record: Incident) => (
        <Space>
          {record.status === 'open' && (
            <Button size="small" icon={<CheckOutlined />} onClick={() => handleAcknowledge(record.id)}>
              确认
            </Button>
          )}
          {record.status !== 'resolved' && record.status !== 'closed' && (
            <Button size="small" type="primary" icon={<CheckCircleOutlined />} onClick={() => handleResolve(record.id)}>
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
      title: '严重程度', dataIndex: 'severity', key: 'severity', width: 100,
      render: (s: string) => <Tag color={severityColor(s)}>{s}</Tag>,
    },
    { title: '标题', dataIndex: 'title', key: 'title', ellipsis: true },
    { title: '来源', dataIndex: 'source', key: 'source', width: 120 },
    { title: '主机', dataIndex: 'host', key: 'host', width: 120 },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 90,
      render: (s: string) => <Tag color={s === 'firing' ? 'red' : 'green'}>{s}</Tag>,
    },
    { title: '触发时间', dataIndex: 'fired_at', key: 'fired_at', width: 170 },
  ]

  const ruleColumns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
    { title: '规则名称', dataIndex: 'name', key: 'name', ellipsis: true },
    {
      title: '类型', dataIndex: 'rule_type', key: 'rule_type', width: 120,
      render: (t: string) => <Tag>{ruleTypeLabel[t] || t}</Tag>,
    },
    {
      title: '严重程度', dataIndex: 'severity', key: 'severity', width: 100,
      render: (s: string) => <Tag color={severityColor(s)}>{s}</Tag>,
    },
    {
      title: '状态', dataIndex: 'enabled', key: 'enabled', width: 90,
      render: (v: boolean, record: AlertRule) => (
        <Switch size="small" checked={v} onChange={(checked) => handleToggleEnabled(record, checked)} />
      ),
    },
    {
      title: '描述', dataIndex: 'description', key: 'description', ellipsis: true,
    },
    { title: '更新时间', dataIndex: 'updated_at', key: 'updated_at', width: 170 },
    {
      title: '操作', key: 'action', width: 120,
      render: (_: any, record: AlertRule) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => openEditRule(record)}>
            编辑
          </Button>
          <Popconfirm title="确定删除此规则？" onConfirm={() => handleDeleteRule(record.id)} okText="删除" cancelText="取消">
            <Button size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 16 }}>
        {tab === 'rules' && (
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreateRule} style={{ marginRight: 8 }}>
            新建规则
          </Button>
        )}
        <Button icon={<ReloadOutlined />} onClick={loadData} loading={loading}>
          刷新
        </Button>
      </div>

      {/* Stats row */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        {tab === 'rules' ? (
          <>
            <Col span={8}>
              <Card size="small">
                <Statistic
                  title="规则总数"
                  value={rules.length}
                  prefix={<SettingOutlined />}
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card size="small">
                <Statistic
                  title="已启用"
                  value={enabledRuleCount}
                  valueStyle={{ color: '#52c41a' }}
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card size="small">
                <Statistic
                  title="已禁用"
                  value={rules.length - enabledRuleCount}
                  valueStyle={{ color: '#999' }}
                />
              </Card>
            </Col>
          </>
        ) : (
          <>
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
                <Statistic title="严重" value={criticalCount} prefix={<CloseCircleOutlined />} valueStyle={{ color: '#ff4d4f' }} />
              </Card>
            </Col>
            <Col span={6}>
              <Card size="small">
                <Statistic title="警告" value={warningCount} prefix={<WarningOutlined />} valueStyle={{ color: '#faad14' }} />
              </Card>
            </Col>
            <Col span={6}>
              <Card size="small">
                <Statistic title="待处理" value={openCount} valueStyle={{ color: openCount > 0 ? '#ff4d4f' : '#52c41a' }} />
              </Card>
            </Col>
          </>
        )}
      </Row>

      {/* Filters (only for incidents/events) */}
      {tab !== 'rules' && (
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
      )}

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
            {
              key: 'rules',
              label: `规则 (${rules.length})`,
              children: (
                <Table
                  dataSource={rules}
                  columns={ruleColumns}
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

      {/* Rule Create/Edit Modal */}
      <Modal
        title={editingRule ? '编辑规则' : '新建规则'}
        open={ruleModalOpen}
        onOk={handleSaveRule}
        onCancel={() => setRuleModalOpen(false)}
        confirmLoading={ruleSaving}
        width={600}
        destroyOnClose
      >
        <Form form={ruleForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="name" label="规则名称" rules={[{ required: true, message: '请输入规则名称' }]}>
            <Input placeholder="例: CPU使用率告警" />
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="rule_type" label="规则类型" rules={[{ required: true }]}>
                <Select
                  options={[
                    { label: '阈值', value: 'threshold' },
                    { label: '异常检测', value: 'anomaly' },
                    { label: '日志模式', value: 'log_pattern' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="severity" label="严重程度">
                <Select
                  options={[
                    { label: 'critical', value: 'critical' },
                    { label: 'warning', value: 'warning' },
                    { label: 'info', value: 'info' },
                  ]}
                />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} placeholder="规则描述" />
          </Form.Item>
          <Form.Item
            name="rule_config"
            label="规则配置 (JSON)"
            rules={[
              {
                validator: (_, value) => {
                  if (!value) return Promise.resolve()
                  try { JSON.parse(value); return Promise.resolve() }
                  catch { return Promise.reject('请输入有效 JSON') }
                },
              },
            ]}
          >
            <Input.TextArea
              rows={4}
              placeholder={'{\n  "metric": "cpu_usage",\n  "threshold": 80,\n  "duration": "5m"\n}'}
            />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item
            name="notify_config"
            label="通知配置 (JSON)"
            rules={[
              {
                validator: (_, value) => {
                  if (!value) return Promise.resolve()
                  try { JSON.parse(value); return Promise.resolve() }
                  catch { return Promise.reject('请输入有效 JSON') }
                },
              },
            ]}
          >
            <Input.TextArea rows={3} placeholder={'{\n  "channels": ["webhook"],\n  "webhook_url": ""\n}'} />
          </Form.Item>
          <Form.Item
            name="silence_config"
            label="静默配置 (JSON)"
            rules={[
              {
                validator: (_, value) => {
                  if (!value) return Promise.resolve()
                  try { JSON.parse(value); return Promise.resolve() }
                  catch { return Promise.reject('请输入有效 JSON') }
                },
              },
            ]}
          >
            <Input.TextArea rows={2} placeholder={'{\n  "duration": "30m"\n}'} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
