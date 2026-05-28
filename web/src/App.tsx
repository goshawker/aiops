import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ConfigProvider, theme } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { useAppStore } from '@/store/app'
import MainLayout from '@/layouts/MainLayout'
import Dashboard from '@/pages/Dashboard'
import Metrics from '@/pages/Metrics'
import Logs from '@/pages/Logs'
import Alerts from '@/pages/Alerts'
import Anomaly from '@/pages/Anomaly'
import RCA from '@/pages/RCA'
import Topology from '@/pages/Topology'
import Traces from '@/pages/Traces'
import Jobs from '@/pages/Jobs'
import Admin from '@/pages/Admin'
import Login from '@/pages/Login'

const queryClient = new QueryClient()

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
        </BrowserRouter>
      </ConfigProvider>
    </QueryClientProvider>
  )
}
