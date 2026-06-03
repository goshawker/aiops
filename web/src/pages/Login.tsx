import { useState } from 'react'
import { Card, Form, Input, Button, Typography, message } from 'antd'
import { UserOutlined, LockOutlined, DashboardOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { useAppStore } from '@/store/app'
import { adminApi } from '@/api/admin'

const { Text } = Typography

export default function Login() {
  const navigate = useNavigate()
  const { setAuth } = useAppStore()
  const [loading, setLoading] = useState(false)

  const handleLogin = async (values: { username: string; password: string }) => {
    setLoading(true)
    try {
      const res = await adminApi.login(values.username, values.password)
      // Store token for API client
      localStorage.setItem('token', res.token)
      setAuth(res.token, { username: res.username, role: res.role, userId: res.user_id, tenantId: res.tenant_id })
      message.success('登录成功')
      navigate('/dashboard')
    } catch (e: any) {
      // Demo mode - auto login when backend unavailable
      const isNetworkError = !e?.status && !e?.error
      if (isNetworkError) {
        message.warning('后端不可用，已进入演示模式')
        setAuth('demo-token', { username: values.username, role: 'admin' })
        navigate('/dashboard')
      } else {
        message.error(e?.error || e?.message || '登录失败，请检查用户名和密码')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'linear-gradient(135deg, #0c1e3a 0%, #1a3a6b 50%, #0c1e3a 100%)',
        position: 'relative',
        overflow: 'hidden',
      }}
    >
      {/* Decorative background elements */}
      <div style={{ position: 'absolute', top: '10%', left: '5%', width: 300, height: 300, borderRadius: '50%', background: 'radial-gradient(circle, rgba(22,119,255,0.08) 0%, transparent 70%)' }} />
      <div style={{ position: 'absolute', bottom: '15%', right: '10%', width: 400, height: 400, borderRadius: '50%', background: 'radial-gradient(circle, rgba(22,119,255,0.06) 0%, transparent 70%)' }} />

      <Card
        style={{
          width: 420,
          borderRadius: 12,
          boxShadow: '0 8px 32px rgba(0,0,0,0.3)',
          position: 'relative',
          zIndex: 1,
        }}
        styles={{ body: { padding: '40px 32px 32px' } }}
      >
        {/* Logo area */}
        <div style={{ textAlign: 'center', marginBottom: 36 }}>
          <div
            style={{
              width: 64,
              height: 64,
              borderRadius: 16,
              background: 'linear-gradient(135deg, #1677ff, #0958d9)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              margin: '0 auto 16px',
              boxShadow: '0 4px 12px rgba(22,119,255,0.3)',
            }}
          >
            <DashboardOutlined style={{ fontSize: 32, color: '#fff' }} />
          </div>
          <div style={{ fontSize: 24, fontWeight: 700, color: '#1a1a1a', letterSpacing: 1 }}>AIOps</div>
          <Text type="secondary" style={{ fontSize: 14, marginTop: 4, display: 'block' }}>智能运维管理平台</Text>
        </div>

        <Form onFinish={handleLogin} size="large">
          <Form.Item name="username" rules={[{ required: true, message: '请输入用户名' }]}>
            <Input
              prefix={<UserOutlined style={{ color: '#bfbfbf' }} />}
              placeholder="用户名"
              style={{ borderRadius: 8, height: 48 }}
            />
          </Form.Item>
          <Form.Item name="password" rules={[{ required: true, message: '请输入密码' }]}>
            <Input.Password
              prefix={<LockOutlined style={{ color: '#bfbfbf' }} />}
              placeholder="密码"
              style={{ borderRadius: 8, height: 48 }}
            />
          </Form.Item>
          <Form.Item style={{ marginBottom: 16 }}>
            <Button
              type="primary"
              htmlType="submit"
              loading={loading}
              block
              style={{ height: 48, borderRadius: 8, fontSize: 16, fontWeight: 500 }}
            >
              登 录
            </Button>
          </Form.Item>
        </Form>

        <div style={{ textAlign: 'center' }}>
          <Text type="secondary" style={{ fontSize: 13 }}>
            演示账号: admin / admin123
          </Text>
        </div>

        {/* Footer info */}
        <div style={{ textAlign: 'center', marginTop: 32 }}>
          <Text type="secondary" style={{ fontSize: 12 }}>
            AIOps v1.0 · 智能运维平台
          </Text>
        </div>
      </Card>
    </div>
  )
}
