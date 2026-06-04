import { useMemo } from 'react'
import { Breadcrumb } from 'antd'
import { useLocation } from 'react-router-dom'
import {
  DashboardOutlined,
  LineChartOutlined,
  FileTextOutlined,
  BranchesOutlined,
  ClusterOutlined,
  BugOutlined,
  ApartmentOutlined,
  AlertOutlined,
  ClockCircleOutlined,
  SettingOutlined,
} from '@ant-design/icons'

const routeMap: Record<string, { icon: React.ReactNode; label: string }> = {
  '/dashboard': { icon: <DashboardOutlined />, label: '仪表盘' },
  '/metrics': { icon: <LineChartOutlined />, label: '指标监控' },
  '/logs': { icon: <FileTextOutlined />, label: '日志管理' },
  '/traces': { icon: <BranchesOutlined />, label: '链路追踪' },
  '/topology': { icon: <ClusterOutlined />, label: '拓扑视图' },
  '/anomaly': { icon: <BugOutlined />, label: '异常检测' },
  '/rca': { icon: <ApartmentOutlined />, label: '根因分析' },
  '/alerts': { icon: <AlertOutlined />, label: '告警管理' },
  '/jobs': { icon: <ClockCircleOutlined />, label: '作业管理' },
  '/admin': { icon: <SettingOutlined />, label: '管理配置' },
}

const moduleGroups: Record<string, { label: string; icon: React.ReactNode }> = {
  dashboard: { label: '总览', icon: <DashboardOutlined /> },
  metrics: { label: '可观测性', icon: <LineChartOutlined /> },
  logs: { label: '可观测性', icon: <FileTextOutlined /> },
  traces: { label: '可观测性', icon: <BranchesOutlined /> },
  topology: { label: '可观测性', icon: <ClusterOutlined /> },
  anomaly: { label: 'VigilOps', icon: <BugOutlined /> },
  rca: { label: 'VigilOps', icon: <ApartmentOutlined /> },
  alerts: { label: 'VigilOps', icon: <AlertOutlined /> },
  jobs: { label: '自动化', icon: <ClockCircleOutlined /> },
  admin: { label: '管理', icon: <SettingOutlined /> },
}

export default function AppBreadcrumb() {
  const location = useLocation()
  const pathname = location.pathname

  const items = useMemo(() => {
    const crumbs: { key: string; title: React.ReactNode }[] = []
    const base = pathname.split('/')[1] || 'dashboard'

    // Module group
    const group = moduleGroups[base]
    if (group) {
      crumbs.push({
        key: `/dashboard`,
        title: <span>{group.icon} {group.label}</span>,
      })
    }

    // Current page
    const page = routeMap[pathname]
    if (page) {
      crumbs.push({
        key: pathname,
        title: <span>{page.icon} {page.label}</span>,
      })
    }

    return crumbs
  }, [pathname])

  return (
    <Breadcrumb
      items={items}
      style={{ marginBottom: 16, fontSize: 13 }}
    />
  )
}
