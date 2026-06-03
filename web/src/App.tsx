import { lazy, Suspense } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ConfigProvider, theme, Spin } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { useAppStore } from '@/store/app'
import MainLayout from '@/layouts/MainLayout'

// Lazy load all page components for code splitting
const Dashboard = lazy(() => import('@/pages/Dashboard'))
const Metrics = lazy(() => import('@/pages/Metrics'))
const Logs = lazy(() => import('@/pages/Logs'))
const Alerts = lazy(() => import('@/pages/Alerts'))
const Anomaly = lazy(() => import('@/pages/Anomaly'))
const RCA = lazy(() => import('@/pages/RCA'))
const Topology = lazy(() => import('@/pages/Topology'))
const Traces = lazy(() => import('@/pages/Traces'))
const Jobs = lazy(() => import('@/pages/Jobs'))
const Admin = lazy(() => import('@/pages/Admin'))
const Login = lazy(() => import('@/pages/Login'))

const queryClient = new QueryClient()

function PageLoading() {
  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '50vh' }}>
      <Spin size="large" />
    </div>
  )
}

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const token = useAppStore((s) => s.token)
  return token ? <>{children}</> : <Navigate to="/login" replace />
}

export default function App() {
  const appTheme = useAppStore((s) => s.theme)

  return (
    <QueryClientProvider client={queryClient}>
      <ConfigProvider
        locale={zhCN}
        theme={{
          algorithm: appTheme === 'dark' ? theme.darkAlgorithm : theme.defaultAlgorithm,
          token: { colorPrimary: '#1677ff' },
        }}
      >
        <BrowserRouter>
          <Suspense fallback={<PageLoading />}>
            <Routes>
              <Route path="/login" element={<Login />} />
              <Route
                path="/"
                element={
                  <PrivateRoute>
                    <MainLayout />
                  </PrivateRoute>
                }
              >
                <Route index element={<Navigate to="/dashboard" replace />} />
                <Route path="dashboard" element={<Dashboard />} />
                <Route path="metrics" element={<Metrics />} />
                <Route path="logs" element={<Logs />} />
                <Route path="alerts" element={<Alerts />} />
                <Route path="anomaly" element={<Anomaly />} />
                <Route path="rca" element={<RCA />} />
                <Route path="topology" element={<Topology />} />
                <Route path="traces" element={<Traces />} />
                <Route path="jobs" element={<Jobs />} />
                <Route path="admin" element={<Admin />} />
              </Route>
            </Routes>
          </Suspense>
        </BrowserRouter>
      </ConfigProvider>
    </QueryClientProvider>
  )
}
