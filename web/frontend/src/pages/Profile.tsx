import { Card, Form, Input, Button, Avatar, Row, Col, Divider, message } from 'antd'
import { UserOutlined, MailOutlined, PhoneOutlined, LockOutlined } from '@ant-design/icons'
import { useAuthStore } from '@/stores/authStore'

export default function Profile() {
  const { user } = useAuthStore()
  const [form] = Form.useForm()

  const handleUpdateProfile = async (values: any) => {
    try {
      console.log('Updating profile:', values)
      // TODO: Replace with actual API call
      message.success('Profile updated successfully!')
    } catch (error) {
      message.error('Failed to update profile')
    }
  }

  const handleChangePassword = async (values: any) => {
    try {
      console.log('Changing password:', values)
      // TODO: Replace with actual API call
      message.success('Password changed successfully!')
      form.resetFields(['currentPassword', 'newPassword', 'confirmPassword'])
    } catch (error) {
      message.error('Failed to change password')
    }
  }

  return (
    <div style={{ width: '100%' }}>
      <div style={{ marginBottom: 24 }}>
        <h1 style={{ margin: 0, fontSize: 24, fontWeight: 600 }}>Profile</h1>
        <p style={{ margin: '8px 0 0', color: '#666' }}>Manage your account settings</p>
      </div>

      <Row gutter={[24, 24]}>
        <Col xs={24} lg={8}>
          <Card>
            <div style={{ textAlign: 'center' }}>
              <Avatar
                size={120}
                icon={<UserOutlined />}
                style={{ backgroundColor: '#1890ff', marginBottom: 16 }}
              />
              <h2 style={{ margin: '16px 0 8px' }}>{user?.username || 'Admin'}</h2>
              <p style={{ color: '#666', margin: 0 }}>{user?.role || 'Administrator'}</p>
              <Divider />
              <div style={{ textAlign: 'left' }}>
                <p style={{ margin: '8px 0', color: '#666' }}>
                  <strong>User ID:</strong> {user?.id || '1'}
                </p>
                <p style={{ margin: '8px 0', color: '#666' }}>
                  <strong>Role:</strong> {user?.role || 'admin'}
                </p>
                <p style={{ margin: '8px 0', color: '#666' }}>
                  <strong>Status:</strong> <span style={{ color: '#52c41a' }}>Active</span>
                </p>
              </div>
            </div>
          </Card>
        </Col>

        <Col xs={24} lg={16}>
          <Card title="Personal Information" style={{ marginBottom: 24 }}>
            <Form
              layout="vertical"
              initialValues={{
                username: user?.username || 'admin',
                email: 'admin@reignx.com',
                phone: '',
                fullName: 'Administrator',
              }}
              onFinish={handleUpdateProfile}
            >
              <Row gutter={16}>
                <Col xs={24} md={12}>
                  <Form.Item
                    label="Username"
                    name="username"
                    rules={[{ required: true, message: 'Please input username' }]}
                  >
                    <Input prefix={<UserOutlined />} disabled />
                  </Form.Item>
                </Col>
                <Col xs={24} md={12}>
                  <Form.Item
                    label="Full Name"
                    name="fullName"
                    rules={[{ required: true, message: 'Please input full name' }]}
                  >
                    <Input prefix={<UserOutlined />} />
                  </Form.Item>
                </Col>
              </Row>

              <Row gutter={16}>
                <Col xs={24} md={12}>
                  <Form.Item
                    label="Email"
                    name="email"
                    rules={[
                      { required: true, message: 'Please input email' },
                      { type: 'email', message: 'Please input valid email' }
                    ]}
                  >
                    <Input prefix={<MailOutlined />} />
                  </Form.Item>
                </Col>
                <Col xs={24} md={12}>
                  <Form.Item
                    label="Phone"
                    name="phone"
                  >
                    <Input prefix={<PhoneOutlined />} />
                  </Form.Item>
                </Col>
              </Row>

              <Form.Item>
                <Button type="primary" htmlType="submit">
                  Update Profile
                </Button>
              </Form.Item>
            </Form>
          </Card>

          <Card title="Change Password">
            <Form
              form={form}
              layout="vertical"
              onFinish={handleChangePassword}
            >
              <Form.Item
                label="Current Password"
                name="currentPassword"
                rules={[{ required: true, message: 'Please input current password' }]}
              >
                <Input.Password prefix={<LockOutlined />} />
              </Form.Item>

              <Form.Item
                label="New Password"
                name="newPassword"
                rules={[
                  { required: true, message: 'Please input new password' },
                  { min: 8, message: 'Password must be at least 8 characters' }
                ]}
              >
                <Input.Password prefix={<LockOutlined />} />
              </Form.Item>

              <Form.Item
                label="Confirm New Password"
                name="confirmPassword"
                dependencies={['newPassword']}
                rules={[
                  { required: true, message: 'Please confirm password' },
                  ({ getFieldValue }) => ({
                    validator(_, value) {
                      if (!value || getFieldValue('newPassword') === value) {
                        return Promise.resolve()
                      }
                      return Promise.reject(new Error('Passwords do not match'))
                    },
                  }),
                ]}
              >
                <Input.Password prefix={<LockOutlined />} />
              </Form.Item>

              <Form.Item>
                <Button type="primary" htmlType="submit">
                  Change Password
                </Button>
              </Form.Item>
            </Form>
          </Card>
        </Col>
      </Row>
    </div>
  )
}
