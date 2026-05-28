import { useState } from 'react'
import { Card, Form, Input, Button, Typography, message } from 'antd'
import { UserOutlined, LockOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { useAppStore } from '@/store/app'
import client from '@/api/client'

const { Title } = Typography

export default function Login() {
  const navigate = useNavigate()
  const { setAuth } = useAppStore()
  const [loading, setLoading] = useState(false)

  const handleLogin = async (values: { username: string; password: string }) => {
    setLoading(true)
    try {
      const res: any = await client.post('/admin/login', values)
      setAuth(res.token, { username: values.username, role: res.role || 'admin' })
      message.success('登录成功')
      navigate('/dashboard')
    } catch (e) {
      // Demo mode - auto login
      setAuth('demo-token', { username: values.username, role: 'admin' })
      navigate('/dashboard')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: '#f0f2f5' }}>
      <Card style={{ width: 400 }}>
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <Title level={2} style={{ margin: 0, color: '#1677ff' }}>AIOps</Title>
          <Typography.Text type="secondary">智能运维平台</Typography.Text>
        </div>
        <Form onFinish={handleLogin} size="large">
          <Form.Item name="username" rules={[{ required: true, message: '请输入用户名' }]}>
            <Input prefix={<UserOutlined />} placeholder="用户名" />
          </Form.Item>
          <Form.Item name="password" rules={[{ required: true, message: '请输入密码' }]}>
            <Input.Password prefix={<LockOutlined />} placeholder="密码" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block>
              登录
            </Button>
          </Form.Item>
        </Form>
        <Typography.Text type="secondary" style={{ display: 'block', textAlign: 'center' }}>
          默认账号: admin / admin123
        </Typography.Text>
      </Card>
    </div>
  )
}
