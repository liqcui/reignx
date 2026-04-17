import { Form, Input, Button, Card, Typography, message } from 'antd'
import { UserOutlined, LockOutlined } from '@ant-design/icons'
import { useAuthStore } from '@/stores/authStore'

const { Title } = Typography

export default function Login() {
  const { login, isLoading } = useAuthStore()
  const [form] = Form.useForm()

  const handleSubmit = async (values: { username: string; password: string }) => {
    try {
      await login(values.username, values.password)
      message.success('Login successful! Redirecting...')
    } catch (error: any) {
      const errorMessage = error?.response?.data?.error || error?.message || 'Login failed. Please check your credentials.'
      message.error(errorMessage)
    }
  }

  return (
    <div
      style={{
        minHeight: '100vh',
        width: '100%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
        padding: '20px',
      }}
    >
      <Card
        style={{
          width: '100%',
          maxWidth: 420,
          boxShadow: '0 10px 40px rgba(0,0,0,0.2)',
          borderRadius: 12,
        }}
        bodyStyle={{ padding: '40px' }}
      >
        <div style={{ textAlign: 'center', marginBottom: 40 }}>
          <div style={{
            width: 64,
            height: 64,
            margin: '0 auto 16px',
            background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
            borderRadius: 12,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: 32,
            color: '#fff',
            fontWeight: 'bold',
          }}>
            RX
          </div>
          <Title level={2} style={{ margin: '0 0 8px' }}>ReignX</Title>
          <Typography.Text type="secondary" style={{ fontSize: 14 }}>
            Distributed Server Management Platform
          </Typography.Text>
        </div>
        <Form form={form} onFinish={handleSubmit} size="large" layout="vertical">
          <Form.Item
            label="Username"
            name="username"
            rules={[{ required: true, message: 'Please input your username!' }]}
          >
            <Input
              prefix={<UserOutlined style={{ color: '#999' }} />}
              placeholder="Enter your username"
              style={{ borderRadius: 8 }}
            />
          </Form.Item>
          <Form.Item
            label="Password"
            name="password"
            rules={[{ required: true, message: 'Please input your password!' }]}
          >
            <Input.Password
              prefix={<LockOutlined style={{ color: '#999' }} />}
              placeholder="Enter your password"
              style={{ borderRadius: 8 }}
            />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0 }}>
            <Button
              type="primary"
              htmlType="submit"
              block
              size="large"
              loading={isLoading}
              style={{
                height: 48,
                borderRadius: 8,
                fontSize: 16,
                fontWeight: 500,
                marginTop: 8,
              }}
            >
              {isLoading ? 'Signing in...' : 'Sign In'}
            </Button>
          </Form.Item>
        </Form>
        <div style={{
          marginTop: 24,
          padding: 12,
          background: '#f5f5f5',
          borderRadius: 8,
          textAlign: 'center',
        }}>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            🔐 Default: <strong>admin</strong> / <strong>admin123</strong>
          </Typography.Text>
        </div>
      </Card>
    </div>
  )
}
