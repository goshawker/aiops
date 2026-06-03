import { useState, useEffect, useCallback } from 'react'
import { Card, Tabs, Table, Tag, Button, Space, Form, Input, Select, Switch, InputNumber, Typography, message, Modal, Popconfirm, Radio, Row, Col } from 'antd'
import {
  UserOutlined, DesktopOutlined, SettingOutlined, FileTextOutlined, SafetyOutlined, RobotOutlined,
  PlusOutlined, DeleteOutlined, ReloadOutlined,
} from '@ant-design/icons'
import { useModelConfigStore } from '@/store/modelConfig'
import { collectorsApi } from '@/api/collectors'
import { adminApi } from '@/api/admin'
import type { Collector } from '@/api/collectors'
import type { User, AuditLog } from '@/api/admin'

const { Text } = Typography

// ── Model Config Panel ─────────────────────────────────────────
function ModelConfigPanel() {
  const { config, updateConfig, resetConfig, getModelLabel } = useModelConfigStore()
  const [testing, setTesting] = useState(false)

  const handleTest = async () => {
    setTesting(true)
    try {
      const endpoint = config.provider === 'ollama' ? config.ollamaEndpoint : config.apiEndpoint
      const resp = await fetch(`${endpoint.replace(/\/+$/, '')}/api/tags`, { signal: AbortSignal.timeout(5000) })
      if (resp.ok) {
        updateConfig({ connected: true, lastTestTime: new Date().toLocaleString('zh-CN') })
        message.success('连接成功')
      } else {
        message.error('连接失败')
      }
    } catch {
      message.error('连接失败，请检查后端服务是否可用')
    } finally {
      setTesting(false)
    }
  }

  return (
    <div style={{ padding: 24, maxWidth: 720 }}>
      <Card size="small" style={{ marginBottom: 24 }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Space size={16}>
            <RobotOutlined style={{ fontSize: 32, color: config.connected ? '#52c41a' : '#999' }} />
            <div>
              <Text strong style={{ fontSize: 16 }}>{getModelLabel()}</Text>
              <div style={{ marginTop: 2 }}>
                <Tag color={config.connected ? 'green' : 'default'}>{config.connected ? '已连接' : '未连接'}</Tag>
                {config.lastTestTime && <Text type="secondary" style={{ fontSize: 12, marginLeft: 8 }}>上次检测: {config.lastTestTime}</Text>}
              </div>
            </div>
          </Space>
          <Space>
            <Button onClick={handleTest} loading={testing} icon={<ReloadOutlined />}>测试连接</Button>
            <Button onClick={resetConfig}>恢复默认</Button>
          </Space>
        </div>
      </Card>

      <Card title="模型提供商" size="small" style={{ marginBottom: 16 }}>
        <Radio.Group
          value={config.provider}
          onChange={(e) => updateConfig({ provider: e.target.value })}
          style={{ width: '100%' }}
        >
          <Space direction="vertical" style={{ width: '100%' }} size={12}>
            <Radio.Button value="ollama" style={{ width: '100%', height: 48, lineHeight: '48px', paddingLeft: 16 }}>
              <Space><RobotOutlined /> Ollama（本地部署）<Text type="secondary" style={{ fontSize: 12 }}>推荐，资源占用低</Text></Space>
            </Radio.Button>
            <Radio.Button value="openai-compatible" style={{ width: '100%', height: 48, lineHeight: '48px', paddingLeft: 16 }}>
              <Space><SettingOutlined /> OpenAI 兼容 API<Text type="secondary" style={{ fontSize: 12 }}>兼容 OpenAI / 百炼 / DeepSeek 等</Text></Space>
            </Radio.Button>
            <Radio.Button value="local" style={{ width: '100%', height: 48, lineHeight: '48px', paddingLeft: 16 }}>
              <Space><SafetyOutlined /> 本地模型文件<Text type="secondary" style={{ fontSize: 12 }}>GGUF / PyTorch</Text></Space>
            </Radio.Button>
          </Space>
        </Radio.Group>
      </Card>

      <Card title="连接配置" size="small" style={{ marginBottom: 16 }}>
        <Form layout="vertical">
          {config.provider === 'ollama' && (
            <>
              <Form.Item label="Ollama 服务地址">
                <Input value={config.ollamaEndpoint} onChange={(e) => updateConfig({ ollamaEndpoint: e.target.value })} placeholder="http://localhost:11434" />
              </Form.Item>
              <Form.Item label="模型名称">
                <Input value={config.ollamaModel} onChange={(e) => updateConfig({ ollamaModel: e.target.value })} placeholder="qwen2:7b" />
              </Form.Item>
            </>
          )}
          {config.provider === 'openai-compatible' && (
            <>
              <Form.Item label="API 地址">
                <Input value={config.apiEndpoint} onChange={(e) => updateConfig({ apiEndpoint: e.target.value })} placeholder="https://api.openai.com/v1" />
              </Form.Item>
              <Form.Item label="API Key">
                <Input.Password value={config.apiKey} onChange={(e) => updateConfig({ apiKey: e.target.value })} placeholder="sk-..." />
              </Form.Item>
              <Form.Item label="模型名称">
                <Input value={config.apiModel} onChange={(e) => updateConfig({ apiModel: e.target.value })} placeholder="gpt-4o-mini" />
              </Form.Item>
            </>
          )}
          {config.provider === 'local' && (
            <Form.Item label="模型文件路径">
              <Input value={config.localModelPath} onChange={(e) => updateConfig({ localModelPath: e.target.value })} placeholder="/data/models/qwen2-7b-instruct.gguf" />
            </Form.Item>
          )}
        </Form>
      </Card>

      <Card title="模型参数" size="small">
        <Form layout="vertical">
          <Row gutter={16}>
            <Col span={6}>
              <Form.Item label="温度">
                <InputNumber value={config.temperature} onChange={(v) => updateConfig({ temperature: v ?? 0.7 })} min={0} max={2} step={0.1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item label="最大 Token 数">
                <InputNumber value={config.maxTokens} onChange={(v) => updateConfig({ maxTokens: v ?? 2048 })} min={128} max={32768} step={128} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item label="上下文长度">
                <InputNumber value={config.contextLength} onChange={(v) => updateConfig({ contextLength: v ?? 4096 })} min={1024} max={128000} step={1024} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item label="超时（秒）">
                <InputNumber value={config.timeout} onChange={(v) => updateConfig({ timeout: v ?? 120 })} min={10} max={600} step={10} style={{ width: '100%' }} addonAfter="秒" />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Card>
    </div>
  )
}

// ── Constants ──────────────────────────────────────────────────

const roleOptions = [
  { label: '管理员 (admin)', value: 'admin' },
  { label: '运维 (operator)', value: 'operator' },
  { label: '只读 (viewer)', value: 'viewer' },
]

const roleLabel: Record<string, string> = { admin: '管理员', operator: '运维', viewer: '只读' }

const dbOptions = [
  { label: 'PostgreSQL（默认）', value: 'postgres' },
  { label: '达梦 DM8', value: 'dameng' },
  { label: '人大金仓 KingbaseES', value: 'kingbase' },
]

const archOptions = [
  { label: 'x86_64（默认）', value: 'x86' },
  { label: 'ARM64（鲲鹏/飞腾）', value: 'arm64' },
  { label: '龙芯 LoongArch', value: 'loongarch' },
]

// ── Main Component ─────────────────────────────────────────────

export default function Admin() {
  const [activeTab, setActiveTab] = useState('users')
  const [addUserVisible, setAddUserVisible] = useState(false)
  const [addUserLoading, setAddUserLoading] = useState(false)
  const [addUserForm] = Form.useForm()

  // Users
  const [users, setUsers] = useState<User[]>([])
  const [userLoading, setUserLoading] = useState(false)
  const [userTotal, setUserTotal] = useState(0)

  // Audit logs
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([])
  const [auditLoading, setAuditLoading] = useState(false)
  const [auditTotal, setAuditTotal] = useState(0)
  const [auditAction, setAuditAction] = useState<string | undefined>()

  // Agents
  const [agents, setAgents] = useState<Collector[]>([])
  const [agentLoading, setAgentLoading] = useState(false)

  // ── Data loading ─────────────────────────────────────────

  const loadUsers = useCallback(async () => {
    setUserLoading(true)
    try {
      const res = await adminApi.listUsers({ limit: 200 })
      setUsers(res.data || [])
      setUserTotal(res.total || 0)
    } catch { setUsers([]) }
    finally { setUserLoading(false) }
  }, [])

  const loadAuditLogs = useCallback(async (action?: string) => {
    setAuditLoading(true)
    try {
      const res = await adminApi.listAuditLogs({ limit: 200, action })
      setAuditLogs(res.data || [])
      setAuditTotal(res.total || 0)
    } catch { setAuditLogs([]) }
    finally { setAuditLoading(false) }
  }, [])

  const loadAgents = useCallback(async () => {
    setAgentLoading(true)
    try {
      const res = await collectorsApi.list({ limit: 200 })
      setAgents(res.data || [])
    } catch { setAgents([]) }
    finally { setAgentLoading(false) }
  }, [])

  useEffect(() => {
    if (activeTab === 'users') loadUsers()
    if (activeTab === 'agents') loadAgents()
    if (activeTab === 'audit') loadAuditLogs(auditAction)
  }, [activeTab, loadUsers, loadAgents, loadAuditLogs, auditAction])

  // ── User actions ─────────────────────────────────────────

  const handleAddUser = async () => {
    try {
      const values = await addUserForm.validateFields()
      setAddUserLoading(true)
      await adminApi.createUser(values)
      message.success('用户已添加')
      setAddUserVisible(false)
      addUserForm.resetFields()
      loadUsers()
    } catch (e: unknown) {
      const err = e as { errorFields?: unknown; error?: string }
      if (err?.errorFields) return // form validation
      message.error(err?.error || '添加失败')
    } finally {
      setAddUserLoading(false)
    }
  }

  const handleDisableUser = async (user: User) => {
    try {
      await adminApi.updateUser(user.id, { status: 'disabled' })
      message.success('已禁用')
      loadUsers()
    } catch { message.error('操作失败') }
  }

  const handleEnableUser = async (user: User) => {
    try {
      await adminApi.updateUser(user.id, { status: 'active' })
      message.success('已启用')
      loadUsers()
    } catch { message.error('操作失败') }
  }

  // ── Table columns ────────────────────────────────────────

  const userColumns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
    { title: '用户名', dataIndex: 'username', key: 'username', width: 120 },
    { title: '显示名', dataIndex: 'display_name', key: 'display_name', width: 120, render: (n: string) => n || '-' },
    { title: '角色', dataIndex: 'role', key: 'role', width: 110, render: (r: string) => <Tag color={r === 'admin' ? 'red' : r === 'operator' ? 'blue' : 'default'}>{roleLabel[r] || r}</Tag> },
    { title: '邮箱', dataIndex: 'email', key: 'email', width: 180, render: (e: string) => e || '-' },
    { title: '状态', dataIndex: 'status', key: 'status', width: 80, render: (s: string) => <Tag color={s === 'active' ? 'green' : 'red'}>{s === 'active' ? '正常' : '禁用'}</Tag> },
    {
      title: '操作', key: 'action', width: 140,
      render: (_: unknown, record: User) => (
        <Space>
          {record.status === 'active' ? (
            <Popconfirm title="确定禁用该用户？" onConfirm={() => handleDisableUser(record)}>
              <Button size="small" danger icon={<DeleteOutlined />}>禁用</Button>
            </Popconfirm>
          ) : (
            <Button size="small" type="primary" onClick={() => handleEnableUser(record)}>启用</Button>
          )}
        </Space>
      ),
    },
  ]

  const agentColumns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
    { title: '名称', dataIndex: 'name', key: 'name', width: 130 },
    { title: '主机名', dataIndex: 'hostname', key: 'hostname', width: 130 },
    { title: 'IP', dataIndex: 'ip', key: 'ip', width: 120 },
    { title: '版本', dataIndex: 'version', key: 'version', width: 80 },
    { title: '状态', dataIndex: 'status', key: 'status', width: 80, render: (s: string) => <Tag color={s === 'online' ? 'green' : 'red'}>{s === 'online' ? '在线' : '离线'}</Tag> },
    { title: '最后心跳', dataIndex: 'last_heartbeat', key: 'last_heartbeat', width: 170, render: (t: string | null) => t ? new Date(t).toLocaleString('zh-CN') : '-' },
  ]

  const auditColumns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
    { title: '用户', dataIndex: 'username', key: 'username', width: 100 },
    { title: '操作', dataIndex: 'action', key: 'action', width: 100, render: (a: string) => <Tag color={a === 'delete' ? 'red' : a === 'create' ? 'green' : 'blue'}>{a}</Tag> },
    { title: '资源', dataIndex: 'resource', key: 'resource', width: 100 },
    { title: '详情', dataIndex: 'detail', key: 'detail', ellipsis: true },
    { title: 'IP', dataIndex: 'ip', key: 'ip', width: 120 },
    { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 170, render: (t: string) => t ? new Date(t).toLocaleString('zh-CN') : '-' },
  ]

  return (
    <div>
      <Card style={{ padding: 0 }}>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          tabBarStyle={{ paddingLeft: 24, marginBottom: 0 }}
          items={[
            // ── 用户与权限 ──────────────────────────────────
            {
              key: 'users',
              label: <span><UserOutlined /> 用户与权限</span>,
              children: (
                <div style={{ padding: 24 }}>
                  <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 16 }}>
                    <Space>
                      <Button icon={<ReloadOutlined />} onClick={loadUsers}>刷新</Button>
                      <Button type="primary" icon={<PlusOutlined />} onClick={() => setAddUserVisible(true)}>添加用户</Button>
                    </Space>
                  </div>
                  <Table
                    dataSource={users}
                    columns={userColumns}
                    rowKey="id"
                    loading={userLoading}
                    size="small"
                    pagination={{ pageSize: 20, total: userTotal, showTotal: (t) => `共 ${t} 个用户` }}
                  />
                </div>
              ),
            },

            // ── 采集器管理 ──────────────────────────────────
            {
              key: 'agents',
              label: <span><DesktopOutlined /> 采集器管理</span>,
              children: (
                <div style={{ padding: 24 }}>
                  <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 16 }}>
                    <Button icon={<ReloadOutlined />} onClick={loadAgents}>刷新</Button>
                  </div>
                  <Table
                    dataSource={agents}
                    columns={agentColumns}
                    rowKey="id"
                    loading={agentLoading}
                    size="small"
                    pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 个采集器` }}
                  />
                </div>
              ),
            },

            // ── 系统配置 ────────────────────────────────────
            {
              key: 'config',
              label: <span><SettingOutlined /> 系统配置</span>,
              children: (
                <div style={{ padding: 24, maxWidth: 640 }}>
                  <Form layout="vertical">
                    <Form.Item label="数据保留天数" extra="指标和日志数据的保留时间">
                      <Space>
                        <InputNumber min={1} max={365} defaultValue={30} addonAfter="天" style={{ width: 150 }} />
                        <Select defaultValue="30d" style={{ width: 100 }} options={[{ label: '30 天', value: '30d' }, { label: '90 天', value: '90d' }, { label: '180 天', value: '180d' }, { label: '365 天', value: '365d' }]} />
                      </Space>
                    </Form.Item>
                    <Form.Item label="SMTP 服务器" extra="用于发送邮件通知">
                      <Input defaultValue="smtp.example.com" placeholder="smtp.example.com" style={{ width: 300 }} />
                    </Form.Item>
                    <Form.Item label="SMTP 端口">
                      <InputNumber defaultValue={465} min={1} max={65535} style={{ width: 120 }} />
                    </Form.Item>
                    <Form.Item label="通知渠道" extra="启用告警通知渠道">
                      <Space>
                        <Switch defaultChecked /> 邮件
                        <Switch defaultChecked /> 钉钉
                        <Switch /> 企业微信
                        <Switch /> Webhook
                      </Space>
                    </Form.Item>
                    <Form.Item label="OIDC 端点" extra="用于单点登录认证">
                      <Input placeholder="https://oidc.example.com/auth" style={{ width: 400 }} />
                    </Form.Item>
                    <Form.Item>
                      <Space>
                        <Button type="primary" onClick={() => message.success('配置已保存')}>保存设置</Button>
                        <Button>恢复默认</Button>
                      </Space>
                    </Form.Item>
                  </Form>
                </div>
              ),
            },

            // ── 审计日志 ────────────────────────────────────
            {
              key: 'audit',
              label: <span><FileTextOutlined /> 审计日志</span>,
              children: (
                <div style={{ padding: 24 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
                    <Space>
                      <Select
                        placeholder="操作类型"
                        allowClear
                        style={{ width: 130 }}
                        value={auditAction}
                        onChange={(v) => setAuditAction(v)}
                        options={[
                          { label: '登录', value: 'login' },
                          { label: '创建', value: 'create' },
                          { label: '修改', value: 'update' },
                          { label: '删除', value: 'delete' },
                        ]}
                      />
                    </Space>
                    <Button icon={<ReloadOutlined />} onClick={() => loadAuditLogs(auditAction)}>刷新</Button>
                  </div>
                  <Table
                    dataSource={auditLogs}
                    columns={auditColumns}
                    rowKey="id"
                    loading={auditLoading}
                    size="small"
                    pagination={{ pageSize: 20, total: auditTotal, showTotal: (t) => `共 ${t} 条记录` }}
                  />
                </div>
              ),
            },

            // ── 信创设置 ────────────────────────────────────
            {
              key: 'xinchuang',
              label: <span><SafetyOutlined /> 信创设置</span>,
              children: (
                <div style={{ padding: 24, maxWidth: 640 }}>
                  <Form layout="vertical">
                    <Form.Item label="数据库类型" extra="切换需重启服务">
                      <Select defaultValue="postgres" style={{ width: 280 }} options={dbOptions} />
                    </Form.Item>
                    <Form.Item label="CPU 架构" extra="根据部署环境选择">
                      <Select defaultValue="x86" style={{ width: 280 }} options={archOptions} />
                    </Form.Item>
                    <Form.Item label="操作系统" extra="当前检测">
                      <Tag color="blue" style={{ padding: '4px 12px' }}>麒麟 V10（ARM64）</Tag>
                    </Form.Item>
                    <Form.Item label="兼容模式" extra="适配国产浏览器及运行环境">
                      <Space>
                        <Switch defaultChecked /> 国产浏览器兼容模式
                        <Switch /> Kata Containers 安全容器
                      </Space>
                    </Form.Item>
                    <Form.Item>
                      <Space>
                        <Button type="primary" onClick={() => message.success('信创设置已保存')}>保存</Button>
                        <Button>检测环境</Button>
                      </Space>
                    </Form.Item>
                  </Form>
                </div>
              ),
            },

            // ── 模型配置 ────────────────────────────────────
            {
              key: 'model',
              label: <span><RobotOutlined /> 模型配置</span>,
              children: <ModelConfigPanel />,
            },
          ]}
        />
      </Card>

      {/* Add User Modal */}
      <Modal
        title="添加用户"
        open={addUserVisible}
        onCancel={() => { setAddUserVisible(false); addUserForm.resetFields() }}
        onOk={handleAddUser}
        confirmLoading={addUserLoading}
      >
        <Form form={addUserForm} layout="vertical">
          <Form.Item name="username" label="用户名" rules={[{ required: true, message: '请输入用户名' }]}>
            <Input placeholder="请输入用户名" />
          </Form.Item>
          <Form.Item name="display_name" label="显示名">
            <Input placeholder="可选" />
          </Form.Item>
          <Form.Item name="email" label="邮箱">
            <Input placeholder="email@example.com" />
          </Form.Item>
          <Form.Item name="role" label="角色" initialValue="viewer" rules={[{ required: true }]}>
            <Select options={roleOptions} />
          </Form.Item>
          <Form.Item name="password" label="初始密码" rules={[{ required: true, message: '请设置密码' }, { min: 6, message: '至少6位' }]}>
            <Input.Password placeholder="设置初始密码" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
