import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { Layout, Menu, theme, Avatar, Dropdown, Space } from 'antd'
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
} from '@ant-design/icons'
import { useAppStore } from '@/store/app'

const { Header, Sider, Content } = Layout

const menuItems = [
  { key: '/dashboard', icon: <DashboardOutlined />, label: '仪表盘' },
  { key: '/metrics', icon: <LineChartOutlined />, label: '指标监控' },
  { key: '/logs', icon: <FileTextOutlined />, label: '日志管理' },
  { key: '/alerts', icon: <AlertOutlined />, label: '告警管理' },
  { key: '/anomaly', icon: <BugOutlined />, label: '异常检测' },
  { key: '/rca', icon: <ApartmentOutlined />, label: '根因分析' },
  { key: '/topology', icon: <ClusterOutlined />, label: '拓扑视图' },
  { key: '/traces', icon: <BranchesOutlined />, label: '链路追踪' },
  { key: '/jobs', icon: <ClockCircleOutlined />, label: '作业管理' },
  { key: '/admin', icon: <SettingOutlined />, label: '管理配置' },
]

export default function MainLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { sidebarCollapsed, toggleSidebar, logout } = useAppStore()
  const { token: { colorBgContainer, borderRadiusLG } } = theme.useToken()

  const userMenuItems = [
    { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: () => { logout(); navigate('/login') } },
  ]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        trigger={null}
        collapsible
        collapsed={sidebarCollapsed}
        theme="light"
        style={{ borderRight: '1px solid #f0f0f0' }}
      >
        <div style={{ height: 64, display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: sidebarCollapsed ? 16 : 20, color: '#1677ff' }}>
          {sidebarCollapsed ? 'A' : 'AIOps'}
        </div>
        <Menu
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout>
        <Header style={{ padding: '0 24px', background: colorBgContainer, display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid #f0f0f0' }}>
          <div onClick={toggleSidebar} style={{ cursor: 'pointer', fontSize: 18 }}>
            {sidebarCollapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
          </div>
          <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
            <Space style={{ cursor: 'pointer' }}>
              <Avatar icon={<UserOutlined />} />
              <span>admin</span>
            </Space>
          </Dropdown>
        </Header>
        <Content style={{ margin: 24, padding: 24, background: colorBgContainer, borderRadius: borderRadiusLG, minHeight: 280 }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}
