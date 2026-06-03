import { useEffect, useMemo, useState } from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import {
  Layout, Menu, Avatar, Dropdown, Space, Badge, Typography,
  Input, Modal, Select, Tooltip, Button, Drawer, Tag,
} from 'antd'
import type { MenuProps } from 'antd'
import {
  DashboardOutlined,
  LineChartOutlined,
  FileTextOutlined,
  AlertOutlined,
  BugOutlined,
  ApartmentOutlined,
  ClusterOutlined,
  BranchesOutlined,
  ClockCircleOutlined,
  SettingOutlined,
  UserOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  LogoutOutlined,
  SearchOutlined,
  BellOutlined,
  SunOutlined,
  MoonOutlined,
  MessageOutlined,
  FundOutlined,
} from '@ant-design/icons'
import { useAppStore } from '@/store/app'
import { useModelConfigStore } from '@/store/modelConfig'
import AppBreadcrumb from '@/components/AppBreadcrumb'

const { Header, Sider, Content } = Layout
const { Text } = Typography

// ── Route / module config ──────────────────────────────────────
interface ModuleGroup {
  key: string
  label: string
  icon: React.ReactNode
  children: { key: string; icon: React.ReactNode; label: string }[]
}

const moduleGroups: ModuleGroup[] = [
  {
    key: 'dashboard',
    label: '总览',
    icon: <DashboardOutlined />,
    children: [{ key: '/dashboard', icon: <DashboardOutlined />, label: '仪表盘' }],
  },
  {
    key: 'observability',
    label: '可观测性',
    icon: <LineChartOutlined />,
    children: [
      { key: '/metrics', icon: <LineChartOutlined />, label: '指标监控' },
      { key: '/logs', icon: <FileTextOutlined />, label: '日志管理' },
      { key: '/traces', icon: <BranchesOutlined />, label: '链路追踪' },
      { key: '/topology', icon: <ClusterOutlined />, label: '拓扑视图' },
    ],
  },
  {
    key: 'aiops',
    label: 'AIOps',
    icon: <BugOutlined />,
    children: [
      { key: '/anomaly', icon: <BugOutlined />, label: '异常检测' },
      { key: '/rca', icon: <ApartmentOutlined />, label: '根因分析' },
      { key: '/alerts', icon: <AlertOutlined />, label: '告警管理' },
    ],
  },
  {
    key: 'automation',
    label: '自动化',
    icon: <ClockCircleOutlined />,
    children: [{ key: '/jobs', icon: <ClockCircleOutlined />, label: '作业管理' }],
  },
  {
    key: 'admin',
    label: '管理',
    icon: <SettingOutlined />,
    children: [{ key: '/admin', icon: <SettingOutlined />, label: '管理配置' }],
  },
]

// Map pathname -> module key
const pathToModule = (path: string): string => {
  const base = path.split('/')[1] || 'dashboard'
  if (base === 'dashboard') return 'dashboard'
  if (['metrics', 'logs', 'traces', 'topology'].includes(base)) return 'observability'
  if (['anomaly', 'rca', 'alerts'].includes(base)) return 'aiops'
  if (['jobs', 'workflows'].includes(base)) return 'automation'
  if (['admin'].includes(base)) return 'admin'
  return 'dashboard'
}

// Module tab items for top navigation
const moduleTabs = moduleGroups.map((m) => ({
  key: m.key,
  label: m.label,
  icon: m.icon,
}))

const firstRouteInModule = (moduleKey: string): string => {
  const group = moduleGroups.find((m) => m.key === moduleKey)
  return group?.children[0]?.key || '/dashboard'
}

// ── Build Ant Design Menu items with group dividers ────────────
const buildMenuItems = (collapsed: boolean): MenuProps['items'] => {
  return moduleGroups.map((group) => ({
    key: `group-${group.key}`,
    type: 'group' as const,
    label: collapsed ? null : group.label,
    children: group.children.map((child) => ({
      key: child.key,
      icon: child.icon,
      label: child.label,
    })),
  }))
}

// ── Global Search Modal ────────────────────────────────────────
function GlobalSearchModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  const navigate = useNavigate()
  const [q, setQ] = useState('')

  const searchOptions = useMemo(() => {
    if (!q) return []
    const all: { key: string; label: string; icon: React.ReactNode; desc: string }[] = [
      { key: '/dashboard', label: '仪表盘', icon: <DashboardOutlined />, desc: '总览页面' },
      { key: '/metrics', label: '指标监控', icon: <LineChartOutlined />, desc: 'PromQL 查询与图表' },
      { key: '/logs', label: '日志管理', icon: <FileTextOutlined />, desc: '日志检索与分析' },
      { key: '/traces', label: '链路追踪', icon: <BranchesOutlined />, desc: '分布式追踪 APM' },
      { key: '/topology', label: '拓扑视图', icon: <ClusterOutlined />, desc: '服务依赖拓扑图' },
      { key: '/anomaly', label: '异常检测', icon: <BugOutlined />, desc: '自动阈值异常检测' },
      { key: '/rca', label: '根因分析', icon: <ApartmentOutlined />, desc: '智能根因定位' },
      { key: '/alerts', label: '告警管理', icon: <AlertOutlined />, desc: '告警降噪与事件管理' },
      { key: '/jobs', label: '作业管理', icon: <ClockCircleOutlined />, desc: '自动化作业' },
      { key: '/admin', label: '管理配置', icon: <SettingOutlined />, desc: '系统设置与权限' },
    ]
    return all.filter(
      (item) =>
        item.label.includes(q) ||
        item.desc.includes(q) ||
        item.key.includes(q),
    )
  }, [q])

  return (
    <Modal
      title={
        <Space>
          <SearchOutlined />
          <span>全局搜索</span>
        </Space>
      }
      open={open}
      onCancel={onClose}
      footer={null}
      width={520}
      destroyOnClose
    >
      <Input
        autoFocus
        value={q}
        onChange={(e) => setQ(e.target.value)}
        placeholder="搜索页面、指标、日志..."
        prefix={<SearchOutlined />}
        size="large"
        style={{ marginBottom: 16 }}
        onPressEnter={() => {
          if (searchOptions.length > 0) {
            navigate(searchOptions[0].key)
            onClose()
            setQ('')
          }
        }}
      />
      {searchOptions.length > 0 && (
        <div style={{ maxHeight: 320, overflow: 'auto' }}>
          {searchOptions.map((opt) => (
            <div
              key={opt.key}
              onClick={() => {
                navigate(opt.key)
                onClose()
                setQ('')
              }}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 12,
                padding: '10px 12px',
                borderRadius: 8,
                cursor: 'pointer',
                transition: 'background 0.15s',
              }}
              onMouseEnter={(e) => (e.currentTarget.style.background = '#f5f5f5')}
              onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
            >
              <span style={{ fontSize: 18, color: '#1677ff' }}>{opt.icon}</span>
              <div>
                <Text strong>{opt.label}</Text>
                <div>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {opt.desc}
                  </Text>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
      {q && searchOptions.length === 0 && (
        <div style={{ textAlign: 'center', padding: 24, color: '#999' }}>未找到匹配结果</div>
      )}
    </Modal>
  )
}

// ── LLM Assistant Drawer ───────────────────────────────────────
function AssistantDrawer({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [messages, setMessages] = useState<{ role: string; content: string }[]>([])
  const [input, setInput] = useState('')
  const { config, getModelLabel } = useModelConfigStore()

  const handleSend = async () => {
    if (!input.trim()) return
    const userMsg = input
    setInput('')
    setMessages((prev) => [...prev, { role: 'user', content: userMsg }])

    // Try to call the configured model endpoint
    const modelLabel = getModelLabel()
    let reply: string

    try {
      if (config.provider === 'ollama') {
        const resp = await fetch(`${config.ollamaEndpoint.replace(/\/+$/, '')}/api/generate`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ model: config.ollamaModel, prompt: userMsg, stream: false, options: { temperature: config.temperature } }),
          signal: AbortSignal.timeout(config.timeout * 1000),
        })
        const data = await resp.json()
        reply = data.response || data.message?.content || `[${modelLabel}] 返回了空响应`
      } else if (config.provider === 'openai-compatible') {
        const resp = await fetch(`${config.apiEndpoint.replace(/\/+$/, '')}/chat/completions`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${config.apiKey}` },
          body: JSON.stringify({ model: config.apiModel, messages: [...messages, { role: 'user', content: userMsg }], temperature: config.temperature, max_tokens: config.maxTokens }),
          signal: AbortSignal.timeout(config.timeout * 1000),
        })
        const data = await resp.json()
        reply = data.choices?.[0]?.message?.content || `[${modelLabel}] 返回了空响应`
      } else {
        reply = `已配置模型: ${modelLabel}\n本地模型文件模式需配合推理服务使用，当前为占位响应。请在「管理 → 模型配置」中切换至 Ollama 或 API 模式。`
      }
    } catch (e: unknown) {
      const err = e as { name?: string; code?: number; message?: string }
      if (err.name === 'TimeoutError' || err.code === 20) {
        reply = `⏱️ 请求超时，请检查 ${modelLabel} 服务是否运行正常`
      } else {
        reply = `⚠️ 连接 ${modelLabel} 失败: ${err.message || '未知错误'}。请在「管理 → 模型配置」中检查连接设置。`
      }
    }

    setMessages((prev) => [...prev, { role: 'assistant', content: reply }])
  }

  return (
    <Drawer
      title={
        <Space>
          <MessageOutlined style={{ color: '#1677ff' }} />
          <span>AI 助手</span>
          <Tag style={{ fontSize: 11, marginLeft: 4 }}>{getModelLabel()}</Tag>
        </Space>
      }
      open={open}
      onClose={onClose}
      width={480}
      footer={
        <Space.Compact style={{ width: '100%' }}>
          <Input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="输入问题，如「过去1小时哪个服务最慢？」"
            onPressEnter={handleSend}
          />
          <Button type="primary" onClick={handleSend}>发送</Button>
        </Space.Compact>
      }
    >
      {messages.length === 0 ? (
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            height: '100%',
            color: '#999',
            textAlign: 'center',
            gap: 8,
          }}
        >
          <MessageOutlined style={{ fontSize: 48, opacity: 0.3 }} />
          <Text type="secondary">你好！我是 AIOps 智能助手</Text>
          <Text type="secondary" style={{ fontSize: 12 }}>
            我可以帮你查询指标、分析日志、排查故障
          </Text>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          {messages.map((msg, i) => (
            <div
              key={i}
              style={{
                alignSelf: msg.role === 'user' ? 'flex-end' : 'flex-start',
                maxWidth: '85%',
                padding: '10px 14px',
                borderRadius: 12,
                background: msg.role === 'user' ? '#1677ff' : '#f5f5f5',
                color: msg.role === 'user' ? '#fff' : '#333',
                fontSize: 14,
                lineHeight: 1.6,
              }}
            >
              {msg.content}
            </div>
          ))}
        </div>
      )}
    </Drawer>
  )
}

// ── Main Layout ────────────────────────────────────────────────
export default function MainLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const pathname = location.pathname
  const {
    sidebarCollapsed, toggleSidebar, logout,
    theme: appTheme, toggleTheme,
    searchVisible, setSearchVisible,
    notificationCount,
    assistantVisible, setAssistantVisible,
    user,
    resetActivity,
  } = useAppStore()

  // Derive current module from pathname
  const currentModule = pathToModule(pathname)

  // ── Top-level module tab click ──────────────────────────────
  const handleModuleTab = (moduleKey: string) => {
    if (moduleKey === currentModule) return
    navigate(firstRouteInModule(moduleKey))
  }

  // ── Keyboard shortcut: ⌘K / Ctrl+K ─────────────────────────
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setSearchVisible(!searchVisible)
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [searchVisible, setSearchVisible])

  // Reset activity on page visibility change (tab switch back)
  useEffect(() => {
    const handler = () => {
      if (document.visibilityState === 'visible') {
        resetActivity()
      }
    }
    document.addEventListener('visibilitychange', handler)
    return () => document.removeEventListener('visibilitychange', handler)
  }, [resetActivity])

  // ── Wrap page components to inject breadcrumb + title ───────
  const pageTitleMap: Record<string, string> = {
    '/dashboard': '仪表盘总览',
    '/metrics': '指标监控',
    '/logs': '日志管理',
    '/traces': '链路追踪',
    '/topology': '拓扑视图',
    '/anomaly': '异常检测',
    '/rca': '根因分析',
    '/alerts': '告警管理',
    '/jobs': '作业管理',
    '/admin': '管理配置',
  }

  // ── User dropdown ───────────────────────────────────────────
  const userMenuItems = [
    { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: () => { logout(); navigate('/login') } },
  ]

  // ── Build Menu items (grouped) ──────────────────────────────
  const menuItems = useMemo(() => buildMenuItems(sidebarCollapsed), [sidebarCollapsed])

  // ── Style helpers ───────────────────────────────────────────
  const isDark = appTheme === 'dark'
  const headerBg = isDark ? '#141414' : '#fff'
  const headerBorder = isDark ? '#303030' : '#f0f0f0'
  const siderBg = isDark ? '#1d1d1d' : '#fff'

  return (
    <Layout style={{ minHeight: '100vh' }}>
      {/* ── Sider ────────────────────────────────────────── */}
      <Sider
        trigger={null}
        collapsible
        collapsed={sidebarCollapsed}
        theme={isDark ? 'dark' : 'light'}
        width={220}
        style={{
          borderRight: `1px solid ${headerBorder}`,
          background: siderBg,
          overflow: 'auto',
          height: '100vh',
          position: 'fixed',
          left: 0,
          top: 56,
          bottom: 0,
          zIndex: 10,
        }}
      >
        {/* Logo area (when sidebar is collapsed / expanded) */}
        <div
          style={{
            height: 56,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontWeight: 700,
            fontSize: sidebarCollapsed ? 20 : 22,
            color: '#1677ff',
            borderBottom: `1px solid ${headerBorder}`,
            letterSpacing: 1,
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <FundOutlined style={{ fontSize: 22, color: '#1677ff' }} />
            {!sidebarCollapsed && <span>AIOps</span>}
          </div>
        </div>

        <Menu
          mode="inline"
          selectedKeys={[pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
          style={{
            borderRight: 0,
            background: 'transparent',
            marginTop: 8,
          }}
        />
      </Sider>

      {/* ── Main area ────────────────────────────────────── */}
      <Layout style={{ marginLeft: sidebarCollapsed ? 64 : 220, transition: 'margin-left 0.2s' }}>
        {/* ── Header ──────────────────────────────────────── */}
        <Header
          style={{
            padding: '0 24px',
            background: headerBg,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            borderBottom: `1px solid ${headerBorder}`,
            height: 56,
            lineHeight: '56px',
            position: 'sticky',
            top: 0,
            zIndex: 20,
          }}
        >
          {/* Left: collapse toggle + module tabs */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, flex: 1, minWidth: 0 }}>
            <Tooltip title={sidebarCollapsed ? '展开菜单' : '收起菜单'}>
              <span
                role="button"
                aria-label="Toggle sidebar"
                tabIndex={0}
                onClick={toggleSidebar}
                onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); toggleSidebar() }}}
                style={{ cursor: 'pointer', fontSize: 18, color: isDark ? '#fff' : '#333', marginRight: 8, flexShrink: 0 }}
              >
                {sidebarCollapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
              </span>
            </Tooltip>

            {/* Module tabs */}
            <div role="tablist" style={{ display: 'flex', gap: 4 }}>
              {moduleTabs.map((tab) => {
                const active = tab.key === currentModule
                return (
                  <div
                    key={tab.key}
                    role="tab"
                    aria-selected={active}
                    tabIndex={0}
                    onClick={() => handleModuleTab(tab.key)}
                    onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); handleModuleTab(tab.key) }}}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 6,
                      padding: '0 16px',
                      height: 40,
                      borderRadius: 8,
                      cursor: 'pointer',
                      fontSize: 14,
                      fontWeight: active ? 600 : 400,
                      color: active ? '#1677ff' : isDark ? '#999' : '#666',
                      background: active ? (isDark ? 'rgba(22,119,255,0.1)' : 'rgba(22,119,255,0.06)') : 'transparent',
                      transition: 'all 0.15s',
                    }}
                  >
                    {tab.icon}
                    <span>{tab.label}</span>
                  </div>
                )
              })}
            </div>
          </div>

          {/* Right: search, time picker, theme, notifications, user */}
          <Space size={12}>
            {/* Global search trigger */}
            <Tooltip title="搜索 (⌘K)">
              <Button
                type="text"
                icon={<SearchOutlined />}
                onClick={() => setSearchVisible(true)}
                aria-label="Search"
                style={{ color: isDark ? '#999' : '#666' }}
              />
            </Tooltip>

            {/* Global time range */}
            <Select
              defaultValue="1h"
              size="small"
              style={{ width: 120 }}
              bordered={false}
              options={[
                { label: '最近 15 分钟', value: '15m' },
                { label: '最近 1 小时', value: '1h' },
                { label: '最近 3 小时', value: '3h' },
                { label: '最近 6 小时', value: '6h' },
                { label: '最近 24 小时', value: '24h' },
                { label: '最近 7 天', value: '7d' },
              ]}
            />

            {/* Theme toggle */}
            <Tooltip title={isDark ? '明亮模式' : '暗黑模式'}>
              <Button
                type="text"
                icon={isDark ? <SunOutlined /> : <MoonOutlined />}
                onClick={toggleTheme}
                aria-label={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
                style={{ color: isDark ? '#999' : '#666' }}
              />
            </Tooltip>

            {/* Notification bell */}
            <Tooltip title="通知">
              <Badge count={notificationCount} size="small">
                <BellOutlined aria-label="Notifications" style={{ fontSize: 16, color: isDark ? '#999' : '#666', cursor: 'pointer' }} />
              </Badge>
            </Tooltip>

            {/* User avatar */}
            <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
              <Space style={{ cursor: 'pointer' }} aria-label="User menu">
                <Avatar size={28} icon={<UserOutlined />} style={{ background: '#1677ff' }} />
                <span style={{ fontSize: 13, color: isDark ? '#ccc' : '#333' }}>{user?.username || 'admin'}</span>
              </Space>
            </Dropdown>
          </Space>
        </Header>

        {/* ── Content ────────────────────────────────────────── */}
        <Content style={{ margin: 0, padding: 24, minHeight: 280, position: 'relative' }}>
          {/* Breadcrumb */}
          <AppBreadcrumb />

          {/* Page title */}
          <Typography.Title level={4} style={{ marginBottom: 20, marginTop: 0 }}>
            {pageTitleMap[pathname] || ''}
          </Typography.Title>

          {/* Page content */}
          <Outlet />
        </Content>
      </Layout>

      {/* ── Global Search Modal ───────────────────────────── */}
      <GlobalSearchModal open={searchVisible} onClose={() => setSearchVisible(false)} />

      {/* ── LLM Assistant Floating Button + Drawer ────────── */}
      <Tooltip title="AI 助手">
        <Button
          type="primary"
          shape="circle"
          size="large"
          icon={<MessageOutlined />}
          onClick={() => setAssistantVisible(true)}
          style={{
            position: 'fixed',
            bottom: 32,
            right: 32,
            width: 52,
            height: 52,
            fontSize: 22,
            boxShadow: '0 4px 14px rgba(22,119,255,0.4)',
            zIndex: 100,
          }}
        />
      </Tooltip>
      <AssistantDrawer open={assistantVisible} onClose={() => setAssistantVisible(false)} />
    </Layout>
  )
}
