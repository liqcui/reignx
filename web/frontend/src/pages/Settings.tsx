import { Card, Form, Switch, Select, Button, Divider, message, Row, Col } from 'antd'
import { BellOutlined, LockOutlined, GlobalOutlined, EyeOutlined } from '@ant-design/icons'

export default function Settings() {
  const [form] = Form.useForm()

  const handleSaveSettings = async (values: any) => {
    try {
      console.log('Saving settings:', values)
      // TODO: Replace with actual API call
      message.success('Settings saved successfully!')
    } catch (error) {
      message.error('Failed to save settings')
    }
  }

  return (
    <div style={{ width: '100%' }}>
      <div style={{ marginBottom: 24 }}>
        <h1 style={{ margin: 0, fontSize: 24, fontWeight: 600 }}>Settings</h1>
        <p style={{ margin: '8px 0 0', color: '#666' }}>Customize your application preferences</p>
      </div>

      <Row gutter={[24, 24]}>
        <Col xs={24} lg={12}>
          <Card title={<><BellOutlined /> Notifications</>} style={{ marginBottom: 24 }}>
            <Form
              form={form}
              layout="vertical"
              initialValues={{
                emailNotifications: true,
                jobNotifications: true,
                taskNotifications: false,
                systemAlerts: true,
              }}
              onFinish={handleSaveSettings}
            >
              <Form.Item
                label="Email Notifications"
                name="emailNotifications"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              <p style={{ marginTop: -16, marginBottom: 16, color: '#666', fontSize: 12 }}>
                Receive email notifications for important events
              </p>

              <Form.Item
                label="Job Completion Notifications"
                name="jobNotifications"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              <p style={{ marginTop: -16, marginBottom: 16, color: '#666', fontSize: 12 }}>
                Get notified when jobs complete
              </p>

              <Form.Item
                label="Task Notifications"
                name="taskNotifications"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              <p style={{ marginTop: -16, marginBottom: 16, color: '#666', fontSize: 12 }}>
                Receive updates for individual task executions
              </p>

              <Form.Item
                label="System Alerts"
                name="systemAlerts"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              <p style={{ marginTop: -16, marginBottom: 16, color: '#666', fontSize: 12 }}>
                Get alerts for system errors and failures
              </p>
            </Form>
          </Card>

          <Card title={<><EyeOutlined /> Appearance</>}>
            <Form
              layout="vertical"
              initialValues={{
                theme: 'light',
                language: 'en',
                dateFormat: 'YYYY-MM-DD',
              }}
            >
              <Form.Item
                label="Theme"
                name="theme"
              >
                <Select>
                  <Select.Option value="light">Light</Select.Option>
                  <Select.Option value="dark">Dark (Coming Soon)</Select.Option>
                  <Select.Option value="auto">Auto (Coming Soon)</Select.Option>
                </Select>
              </Form.Item>

              <Form.Item
                label="Language"
                name="language"
              >
                <Select>
                  <Select.Option value="en">English</Select.Option>
                  <Select.Option value="zh">中文 (Coming Soon)</Select.Option>
                  <Select.Option value="ja">日本語 (Coming Soon)</Select.Option>
                </Select>
              </Form.Item>

              <Form.Item
                label="Date Format"
                name="dateFormat"
              >
                <Select>
                  <Select.Option value="YYYY-MM-DD">YYYY-MM-DD</Select.Option>
                  <Select.Option value="MM/DD/YYYY">MM/DD/YYYY</Select.Option>
                  <Select.Option value="DD/MM/YYYY">DD/MM/YYYY</Select.Option>
                </Select>
              </Form.Item>
            </Form>
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card title={<><LockOutlined /> Security</>} style={{ marginBottom: 24 }}>
            <Form
              layout="vertical"
              initialValues={{
                twoFactorAuth: false,
                sessionTimeout: 60,
                autoLogout: true,
              }}
            >
              <Form.Item
                label="Two-Factor Authentication"
                name="twoFactorAuth"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              <p style={{ marginTop: -16, marginBottom: 16, color: '#666', fontSize: 12 }}>
                Enable two-factor authentication for additional security
              </p>

              <Form.Item
                label="Session Timeout (minutes)"
                name="sessionTimeout"
              >
                <Select>
                  <Select.Option value={15}>15 minutes</Select.Option>
                  <Select.Option value={30}>30 minutes</Select.Option>
                  <Select.Option value={60}>1 hour</Select.Option>
                  <Select.Option value={120}>2 hours</Select.Option>
                  <Select.Option value={480}>8 hours</Select.Option>
                </Select>
              </Form.Item>

              <Form.Item
                label="Auto Logout on Inactivity"
                name="autoLogout"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              <p style={{ marginTop: -16, marginBottom: 16, color: '#666', fontSize: 12 }}>
                Automatically logout after period of inactivity
              </p>
            </Form>
          </Card>

          <Card title={<><GlobalOutlined /> Regional Settings</>}>
            <Form
              layout="vertical"
              initialValues={{
                timezone: 'UTC',
                numberFormat: 'en-US',
              }}
            >
              <Form.Item
                label="Timezone"
                name="timezone"
              >
                <Select showSearch>
                  <Select.Option value="UTC">UTC (GMT+0)</Select.Option>
                  <Select.Option value="America/New_York">Eastern Time (GMT-5)</Select.Option>
                  <Select.Option value="America/Los_Angeles">Pacific Time (GMT-8)</Select.Option>
                  <Select.Option value="Europe/London">London (GMT+0)</Select.Option>
                  <Select.Option value="Asia/Tokyo">Tokyo (GMT+9)</Select.Option>
                  <Select.Option value="Asia/Shanghai">Shanghai (GMT+8)</Select.Option>
                </Select>
              </Form.Item>

              <Form.Item
                label="Number Format"
                name="numberFormat"
              >
                <Select>
                  <Select.Option value="en-US">1,234.56 (US)</Select.Option>
                  <Select.Option value="de-DE">1.234,56 (DE)</Select.Option>
                  <Select.Option value="fr-FR">1 234,56 (FR)</Select.Option>
                </Select>
              </Form.Item>
            </Form>
          </Card>
        </Col>
      </Row>

      <Divider />

      <div style={{ textAlign: 'right' }}>
        <Button style={{ marginRight: 8 }}>
          Reset to Defaults
        </Button>
        <Button type="primary" onClick={() => form.submit()}>
          Save All Settings
        </Button>
      </div>
    </div>
  )
}
