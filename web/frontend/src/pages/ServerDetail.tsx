import { useParams } from 'react-router-dom'
import { Card, Descriptions, Tag, Tabs, Button, Space, Row, Col, Modal, Input, Upload, Select, Form, message, Dropdown } from 'antd'
import { useState, useEffect } from 'react'
import Terminal from '@/components/Terminal/Terminal'
import { serversAPI } from '@/lib/api'
import { useAuthStore } from '@/stores/authStore'
import {
  SyncOutlined,
  PoweroffOutlined,
  ThunderboltOutlined,
  ReloadOutlined,
  UploadOutlined,
  DownOutlined,
  FileOutlined,
  InfoCircleOutlined,
  HddOutlined,
  AppstoreOutlined,
  BugOutlined,
  FileTextOutlined,
  DashboardOutlined,
  DatabaseOutlined,
  CodeOutlined
} from '@ant-design/icons'
import type { MenuProps } from 'antd'

interface ServerDetails {
  id: string
  hostname: string
  ip: string
  os: string
  status: string
  mode: string
  lastSeen: string
  cpuUsage: number
  memoryUsage: number
  diskUsage: number
  uptime: string
  packages: number
  pendingPatches: number
}

export default function ServerDetail() {
  const { id } = useParams<{ id: string }>()
  const { token } = useAuthStore()
  const [server, setServer] = useState<ServerDetails | null>(null)
  const [activeTab, setActiveTab] = useState('overview')
  const [isCommandModalOpen, setIsCommandModalOpen] = useState(false)
  const [isDeployModalOpen, setIsDeployModalOpen] = useState(false)
  const [isInstallOSModalOpen, setIsInstallOSModalOpen] = useState(false)
  const [commandForm] = Form.useForm()
  const [deployForm] = Form.useForm()
  const [installOSForm] = Form.useForm()

  useEffect(() => {
    const fetchServer = async () => {
      if (!id) return

      try {
        const data = await serversAPI.get(id)
        setServer({
          id: data.id,
          hostname: data.hostname,
          ip: data.ip_address || data.ip,
          os: `${data.os} ${data.os_version || ''}`.trim(),
          status: data.status,
          mode: data.mode,
          lastSeen: data.last_seen || 'Unknown',
          cpuUsage: data.cpu_usage || 0,
          memoryUsage: data.memory_usage || 0,
          diskUsage: data.disk_usage || 0,
          uptime: data.uptime ? `${Math.floor(data.uptime / 86400)} days` : 'Unknown',
          packages: 0,
          pendingPatches: 0,
        })
      } catch (error) {
        console.error('Failed to fetch server:', error)
        message.error('Failed to load server details')
      }
    }

    fetchServer()
  }, [id])

  const handlePowerAction = (action: 'reboot' | 'poweroff' | 'poweron') => {
    const actionNames = {
      reboot: 'Reboot',
      poweroff: 'Power Off',
      poweron: 'Power On'
    }

    Modal.confirm({
      title: `Confirm ${actionNames[action]}`,
      content: `Are you sure you want to ${action} server ${server?.hostname}?`,
      okText: 'Yes',
      okType: action === 'poweroff' ? 'danger' : 'primary',
      onOk: async () => {
        try {
          // TODO: Call API endpoint
          message.success(`${actionNames[action]} command sent successfully`)
        } catch (error) {
          message.error(`Failed to ${action} server`)
        }
      },
    })
  }

  const handleExecuteCommand = async (values: any) => {
    try {
      console.log('Executing command:', values)
      // TODO: Call API endpoint
      message.success('Command executed successfully')
      setIsCommandModalOpen(false)
      commandForm.resetFields()
    } catch (error) {
      message.error('Failed to execute command')
    }
  }

  const handleUploadFile = async (values: any) => {
    try {
      if (!id) return

      const formData = new FormData()

      // Get the file from upload component
      const fileList = values.file?.fileList
      if (!fileList || fileList.length === 0) {
        message.error('Please select a file to upload')
        return
      }

      const file = fileList[0].originFileObj
      formData.append('file', file)
      formData.append('targetPath', values.targetPath)

      message.loading({ content: 'Uploading file...', key: 'upload' })

      // Get token from localStorage like axios interceptor does
      const authStorage = localStorage.getItem('auth-storage')
      let authToken = token
      if (authStorage) {
        try {
          const { state } = JSON.parse(authStorage)
          if (state?.token) {
            authToken = state.token
          }
        } catch (error) {
          console.error('Failed to parse auth storage:', error)
        }
      }

      const response = await fetch(`/api/v1/servers/${id}/upload`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${authToken}`,
        },
        body: formData,
      })

      const data = await response.json()

      if (response.ok) {
        message.success({ content: data.message || 'File uploaded successfully', key: 'upload' })
        setIsDeployModalOpen(false)
        deployForm.resetFields()
      } else {
        message.error({ content: data.error || 'Upload failed', key: 'upload' })
      }
    } catch (error) {
      console.error('Upload error:', error)
      message.error({ content: 'Failed to upload file', key: 'upload' })
    }
  }

  const handleInstallOS = async (values: any) => {
    Modal.confirm({
      title: '⚠️ Confirm OS Reinstallation',
      content: (
        <div>
          <p><strong>This will ERASE ALL DATA on the server!</strong></p>
          <p>Server: {server?.hostname}</p>
          <p>OS: {values.osType} {values.osVersion}</p>
          <p>Are you absolutely sure you want to proceed?</p>
        </div>
      ),
      okText: 'Yes, Reinstall OS',
      okType: 'danger',
      cancelText: 'Cancel',
      onOk: async () => {
        try {
          console.log('Installing OS:', values)
          // TODO: Call API endpoint
          message.success('OS installation initiated. Server will reboot and begin PXE boot.')
          setIsInstallOSModalOpen(false)
          installOSForm.resetFields()
        } catch (error) {
          message.error('Failed to start OS installation')
        }
      },
    })
  }

  if (!server) {
    return <div>Loading...</div>
  }

  const statusColor = server.status === 'active' ? 'green' : server.status === 'offline' ? 'red' : 'orange'

  const powerMenuItems: MenuProps['items'] = [
    {
      key: 'reboot',
      label: 'Reboot',
      icon: <ReloadOutlined />,
      onClick: () => handlePowerAction('reboot'),
    },
    {
      key: 'poweroff',
      label: 'Power Off',
      icon: <PoweroffOutlined />,
      danger: true,
      onClick: () => handlePowerAction('poweroff'),
    },
    {
      key: 'poweron',
      label: 'Power On',
      icon: <ThunderboltOutlined />,
      onClick: () => handlePowerAction('poweron'),
    },
  ]

  return (
    <div style={{
      padding: '24px',
      minHeight: '100vh',
      background: 'linear-gradient(135deg, #f5f7fa 0%, #e8edf2 100%)'
    }}>
      <div style={{ maxWidth: 1400, margin: '0 auto' }}>
        {/* Modern Header */}
        <Card
          bordered={false}
          style={{
            marginBottom: 16,
            borderRadius: 10,
            boxShadow: '0 2px 8px rgba(102, 126, 234, 0.15)',
            background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)'
          }}
          bodyStyle={{ padding: '16px 20px' }}
        >
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '16px' }}>
            <div>
              <h2 style={{
                margin: 0,
                fontSize: 22,
                fontWeight: 700,
                color: '#ffffff'
              }}>
                {server.hostname}
              </h2>
              <div style={{ marginTop: 8 }}>
                <Space size={10}>
                  <Tag
                    color={statusColor}
                    style={{
                      margin: 0,
                      borderRadius: 5,
                      padding: '3px 10px',
                      fontSize: 11,
                      fontWeight: 600,
                      border: 'none'
                    }}
                  >
                    {server.status.toUpperCase()}
                  </Tag>
                  <Tag
                    color={server.mode === 'agent' ? 'blue' : 'purple'}
                    style={{
                      margin: 0,
                      borderRadius: 5,
                      padding: '3px 10px',
                      fontSize: 11,
                      fontWeight: 600,
                      border: 'none'
                    }}
                  >
                    {server.mode.toUpperCase()}
                  </Tag>
                  <span style={{
                    color: 'rgba(255, 255, 255, 0.9)',
                    fontSize: 13,
                    fontWeight: 500
                  }}>
                    {server.ip} • {server.lastSeen}
                  </span>
                </Space>
              </div>
            </div>
            <Space size={6}>
              <Button
                danger
                size="middle"
                icon={<SyncOutlined />}
                onClick={() => setIsInstallOSModalOpen(true)}
                style={{
                  borderRadius: 6,
                  fontWeight: 500,
                  fontSize: 12,
                  boxShadow: '0 2px 6px rgba(255, 77, 79, 0.2)',
                  border: 'none',
                  height: 32
                }}
              >
                Reinstall OS
              </Button>
              <Button
                size="middle"
                icon={<UploadOutlined />}
                onClick={() => setIsDeployModalOpen(true)}
                style={{
                  borderRadius: 6,
                  fontWeight: 500,
                  fontSize: 12,
                  background: '#ffffff',
                  borderColor: 'transparent',
                  boxShadow: '0 2px 6px rgba(0, 0, 0, 0.08)',
                  color: '#667eea',
                  height: 32
                }}
              >
                Upload File
              </Button>
              <Dropdown menu={{ items: powerMenuItems }} placement="bottomRight">
                <Button
                  size="middle"
                  icon={<PoweroffOutlined />}
                  style={{
                    borderRadius: 6,
                    fontWeight: 500,
                    fontSize: 12,
                    background: '#ffffff',
                    borderColor: 'transparent',
                    boxShadow: '0 2px 6px rgba(0, 0, 0, 0.08)',
                    color: '#667eea',
                    height: 32
                  }}
                >
                  Power <DownOutlined />
                </Button>
              </Dropdown>
            </Space>
          </div>
        </Card>

        {/* Metrics Cards - Compact Design */}
        <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
          <Col xs={12} sm={12} lg={6}>
            <Card
              bordered={false}
              style={{
                borderRadius: 10,
                background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                boxShadow: '0 2px 8px rgba(102, 126, 234, 0.2)',
                transition: 'all 0.2s ease',
                cursor: 'pointer',
              }}
              bodyStyle={{ padding: '14px 16px' }}
              onMouseEnter={(e) => {
                e.currentTarget.style.transform = 'translateY(-2px)'
                e.currentTarget.style.boxShadow = '0 4px 12px rgba(102, 126, 234, 0.3)'
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.transform = 'translateY(0)'
                e.currentTarget.style.boxShadow = '0 2px 8px rgba(102, 126, 234, 0.2)'
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                <div style={{
                  background: 'rgba(255, 255, 255, 0.15)',
                  width: 44,
                  height: 44,
                  borderRadius: 10,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  flexShrink: 0,
                }}>
                  <DashboardOutlined style={{ fontSize: 20, color: '#ffffff' }} />
                </div>
                <div style={{ flex: 1 }}>
                  <div style={{ color: 'rgba(255, 255, 255, 0.85)', fontSize: 12, fontWeight: 500, marginBottom: 2 }}>
                    CPU
                  </div>
                  <div style={{ color: '#ffffff', fontSize: 24, fontWeight: 700, lineHeight: 1 }}>
                    {server.cpuUsage.toFixed(2)}%
                  </div>
                </div>
              </div>
            </Card>
          </Col>
          <Col xs={12} sm={12} lg={6}>
            <Card
              bordered={false}
              style={{
                borderRadius: 10,
                background: 'linear-gradient(135deg, #1890ff 0%, #096dd9 100%)',
                boxShadow: '0 2px 8px rgba(24, 144, 255, 0.2)',
                transition: 'all 0.2s ease',
                cursor: 'pointer',
              }}
              bodyStyle={{ padding: '14px 16px' }}
              onMouseEnter={(e) => {
                e.currentTarget.style.transform = 'translateY(-2px)'
                e.currentTarget.style.boxShadow = '0 4px 12px rgba(24, 144, 255, 0.3)'
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.transform = 'translateY(0)'
                e.currentTarget.style.boxShadow = '0 2px 8px rgba(24, 144, 255, 0.2)'
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                <div style={{
                  background: 'rgba(255, 255, 255, 0.15)',
                  width: 44,
                  height: 44,
                  borderRadius: 10,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  flexShrink: 0,
                }}>
                  <DatabaseOutlined style={{ fontSize: 20, color: '#ffffff' }} />
                </div>
                <div style={{ flex: 1 }}>
                  <div style={{ color: 'rgba(255, 255, 255, 0.85)', fontSize: 12, fontWeight: 500, marginBottom: 2 }}>
                    Memory
                  </div>
                  <div style={{ color: '#ffffff', fontSize: 24, fontWeight: 700, lineHeight: 1 }}>
                    {server.memoryUsage.toFixed(2)}%
                  </div>
                </div>
              </div>
            </Card>
          </Col>
          <Col xs={12} sm={12} lg={6}>
            <Card
              bordered={false}
              style={{
                borderRadius: 10,
                background: 'linear-gradient(135deg, #52c41a 0%, #389e0d 100%)',
                boxShadow: '0 2px 8px rgba(82, 196, 26, 0.2)',
                transition: 'all 0.2s ease',
                cursor: 'pointer',
              }}
              bodyStyle={{ padding: '14px 16px' }}
              onMouseEnter={(e) => {
                e.currentTarget.style.transform = 'translateY(-2px)'
                e.currentTarget.style.boxShadow = '0 4px 12px rgba(82, 196, 26, 0.3)'
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.transform = 'translateY(0)'
                e.currentTarget.style.boxShadow = '0 2px 8px rgba(82, 196, 26, 0.2)'
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                <div style={{
                  background: 'rgba(255, 255, 255, 0.15)',
                  width: 44,
                  height: 44,
                  borderRadius: 10,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  flexShrink: 0,
                }}>
                  <HddOutlined style={{ fontSize: 20, color: '#ffffff' }} />
                </div>
                <div style={{ flex: 1 }}>
                  <div style={{ color: 'rgba(255, 255, 255, 0.85)', fontSize: 12, fontWeight: 500, marginBottom: 2 }}>
                    Disk
                  </div>
                  <div style={{ color: '#ffffff', fontSize: 24, fontWeight: 700, lineHeight: 1 }}>
                    {server.diskUsage.toFixed(2)}%
                  </div>
                </div>
              </div>
            </Card>
          </Col>
          <Col xs={12} sm={12} lg={6}>
            <Card
              bordered={false}
              style={{
                borderRadius: 10,
                background: 'linear-gradient(135deg, #faad14 0%, #d48806 100%)',
                boxShadow: '0 2px 8px rgba(250, 173, 20, 0.2)',
                transition: 'all 0.2s ease',
                cursor: 'pointer',
              }}
              bodyStyle={{ padding: '14px 16px' }}
              onMouseEnter={(e) => {
                e.currentTarget.style.transform = 'translateY(-2px)'
                e.currentTarget.style.boxShadow = '0 4px 12px rgba(250, 173, 20, 0.3)'
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.transform = 'translateY(0)'
                e.currentTarget.style.boxShadow = '0 2px 8px rgba(250, 173, 20, 0.2)'
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                <div style={{
                  background: 'rgba(255, 255, 255, 0.15)',
                  width: 44,
                  height: 44,
                  borderRadius: 10,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  flexShrink: 0,
                }}>
                  <BugOutlined style={{ fontSize: 20, color: '#ffffff' }} />
                </div>
                <div style={{ flex: 1 }}>
                  <div style={{ color: 'rgba(255, 255, 255, 0.85)', fontSize: 12, fontWeight: 500, marginBottom: 2 }}>
                    Patches
                  </div>
                  <div style={{ color: '#ffffff', fontSize: 24, fontWeight: 700, lineHeight: 1 }}>
                    {server.pendingPatches}
                  </div>
                </div>
              </div>
            </Card>
          </Col>
        </Row>

        {/* Tabs Section - Modern */}
        <Card
          bordered={false}
          style={{
            borderRadius: 12,
            boxShadow: '0 2px 8px rgba(0,0,0,0.08)'
          }}
          bodyStyle={{ padding: '24px' }}
        >
          <Tabs
            activeKey={activeTab}
            onChange={setActiveTab}
            tabBarStyle={{
              marginBottom: 24,
              borderBottom: '2px solid #e5e7eb'
            }}
            items={[
              {
                key: 'overview',
                label: (
                  <span style={{ fontSize: 14, fontWeight: 500 }}>
                    <InfoCircleOutlined style={{ marginRight: 6 }} />
                    Overview
                  </span>
                ),
                children: (
                  <Descriptions
                    bordered
                    column={{ xs: 1, sm: 1, md: 2 }}
                    size="middle"
                    labelStyle={{
                      background: '#f9fafb',
                      fontWeight: 500,
                      color: '#374151'
                    }}
                    contentStyle={{
                      background: '#ffffff'
                    }}
                  >
                    <Descriptions.Item label="Server ID">{server.id}</Descriptions.Item>
                    <Descriptions.Item label="Hostname">{server.hostname}</Descriptions.Item>
                    <Descriptions.Item label="IP Address">{server.ip}</Descriptions.Item>
                    <Descriptions.Item label="Operating System">{server.os}</Descriptions.Item>
                    <Descriptions.Item label="Mode">
                      <Tag
                        color={server.mode === 'agent' ? 'blue' : 'purple'}
                        style={{ borderRadius: 6, fontWeight: 500 }}
                      >
                        {server.mode.toUpperCase()}
                      </Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="Last Seen">{server.lastSeen}</Descriptions.Item>
                    <Descriptions.Item label="Uptime">{server.uptime}</Descriptions.Item>
                    <Descriptions.Item label="Packages">{server.packages}</Descriptions.Item>
                  </Descriptions>
                ),
              },
              {
                key: 'terminal',
                label: (
                  <span style={{ fontSize: 14, fontWeight: 500 }}>
                    <CodeOutlined style={{ marginRight: 6 }} />
                    Terminal
                  </span>
                ),
                children: (
                  <div style={{
                    background: '#1e1e1e',
                    borderRadius: 8,
                    padding: 16,
                    minHeight: 500
                  }}>
                    <Terminal serverId={server.id} />
                  </div>
                ),
              },
              {
                key: 'packages',
                label: (
                  <span style={{ fontSize: 14, fontWeight: 500 }}>
                    <AppstoreOutlined style={{ marginRight: 6 }} />
                    Packages
                  </span>
                ),
                children: (
                  <div style={{
                    padding: 60,
                    textAlign: 'center',
                    color: '#9ca3af'
                  }}>
                    Package list coming soon...
                  </div>
                ),
              },
              {
                key: 'patches',
                label: (
                  <span style={{ fontSize: 14, fontWeight: 500 }}>
                    <BugOutlined style={{ marginRight: 6 }} />
                    Patches
                  </span>
                ),
                children: (
                  <div style={{
                    padding: 60,
                    textAlign: 'center',
                    color: '#9ca3af'
                  }}>
                    Patch list coming soon...
                  </div>
                ),
              },
              {
                key: 'logs',
                label: (
                  <span style={{ fontSize: 14, fontWeight: 500 }}>
                    <FileTextOutlined style={{ marginRight: 6 }} />
                    Logs
                  </span>
                ),
                children: (
                  <div style={{
                    padding: 60,
                    textAlign: 'center',
                    color: '#9ca3af'
                  }}>
                    Server logs coming soon...
                  </div>
                ),
              },
            ]}
          />
        </Card>

        {/* Execute Command Modal */}
        <Modal
        title="Execute Shell Command"
        open={isCommandModalOpen}
        onCancel={() => {
          setIsCommandModalOpen(false)
          commandForm.resetFields()
        }}
        onOk={() => commandForm.submit()}
        width={600}
      >
        <Form form={commandForm} layout="vertical" onFinish={handleExecuteCommand}>
          <Form.Item
            name="command"
            label="Command"
            rules={[{ required: true, message: 'Please enter command' }]}
          >
            <Input.TextArea
              rows={4}
              placeholder="e.g., ls -la /var/log&#10;df -h&#10;systemctl status nginx"
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>

          <Form.Item
            name="mode"
            label="Execution Mode"
            initialValue="ssh"
          >
            <Select>
              <Select.Option value="ssh">SSH (Direct)</Select.Option>
              <Select.Option value="agent">Agent (If available)</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="timeout"
            label="Timeout (seconds)"
            initialValue={30}
          >
            <Select>
              <Select.Option value={10}>10 seconds</Select.Option>
              <Select.Option value={30}>30 seconds</Select.Option>
              <Select.Option value={60}>1 minute</Select.Option>
              <Select.Option value={300}>5 minutes</Select.Option>
              <Select.Option value={600}>10 minutes</Select.Option>
            </Select>
          </Form.Item>

          <div style={{ padding: 12, background: '#fffbe6', border: '1px solid #ffe58f', borderRadius: 4, marginTop: 8 }}>
            <p style={{ margin: 0, fontSize: 13, color: '#ad6800' }}>
              ⚠️ Commands will be executed with root/admin privileges.
            </p>
          </div>
        </Form>
      </Modal>

      {/* Upload File Modal */}
      <Modal
        title="Upload File to Server"
        open={isDeployModalOpen}
        onCancel={() => {
          setIsDeployModalOpen(false)
          deployForm.resetFields()
        }}
        onOk={() => deployForm.submit()}
        width={600}
      >
        <Form form={deployForm} layout="vertical" onFinish={handleUploadFile}>
          <Form.Item
            name="file"
            label="Select File"
            rules={[{ required: true, message: 'Please select a file to upload' }]}
          >
            <Upload.Dragger
              maxCount={1}
              beforeUpload={() => false}
              style={{ padding: '20px' }}
            >
              <p className="ant-upload-drag-icon">
                <UploadOutlined style={{ fontSize: 48, color: '#667eea' }} />
              </p>
              <p className="ant-upload-text" style={{ fontSize: 16, fontWeight: 500 }}>
                Click or drag file to upload
              </p>
              <p className="ant-upload-hint" style={{ fontSize: 13, color: '#8c8c8c' }}>
                Support for single file upload. Any file type accepted.
              </p>
            </Upload.Dragger>
          </Form.Item>

          <Form.Item
            name="targetPath"
            label="Target Path on Server"
            rules={[{ required: true, message: 'Please specify target path' }]}
            tooltip="Full path where the file will be uploaded"
          >
            <Input
              placeholder="e.g., /tmp/myfile.txt or /opt/app/config.yaml"
              prefix={<FileOutlined />}
            />
          </Form.Item>

          <div style={{ padding: 12, background: '#e6f7ff', border: '1px solid #91d5ff', borderRadius: 4 }}>
            <p style={{ margin: 0, fontSize: 13, color: '#0050b3' }}>
              💡 File will be uploaded via SSH/SCP to the specified path on the target server.
            </p>
          </div>
        </Form>
      </Modal>

      {/* Install OS Modal */}
      <Modal
        title="Reinstall Operating System"
        open={isInstallOSModalOpen}
        onCancel={() => {
          setIsInstallOSModalOpen(false)
          installOSForm.resetFields()
        }}
        onOk={() => installOSForm.submit()}
        okText="Install OS"
        okButtonProps={{ danger: true }}
        width={650}
      >
        <Form form={installOSForm} layout="vertical" onFinish={handleInstallOS}>
          <div style={{ padding: 12, background: '#fff1f0', border: '1px solid #ffa39e', borderRadius: 4, marginBottom: 16 }}>
            <p style={{ margin: 0, fontSize: 13, color: '#cf1322' }}>
              <strong>⚠️ Warning:</strong> This will ERASE ALL DATA and reinstall the OS!
            </p>
          </div>

          <Form.Item
            name="osType"
            label="Operating System"
            rules={[{ required: true, message: 'Please select OS type' }]}
          >
            <Select placeholder="Select OS">
              <Select.Option value="ubuntu">Ubuntu</Select.Option>
              <Select.Option value="debian">Debian</Select.Option>
              <Select.Option value="centos">CentOS</Select.Option>
              <Select.Option value="rhel">Red Hat Enterprise Linux</Select.Option>
              <Select.Option value="rocky">Rocky Linux</Select.Option>
              <Select.Option value="alma">AlmaLinux</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="osVersion"
            label="Version"
            rules={[{ required: true, message: 'Please enter OS version' }]}
          >
            <Select placeholder="Select version">
              <Select.Option value="22.04">22.04 (LTS)</Select.Option>
              <Select.Option value="20.04">20.04 (LTS)</Select.Option>
              <Select.Option value="8">8</Select.Option>
              <Select.Option value="9">9</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="rootPassword"
            label="Root Password"
            rules={[
              { required: true, message: 'Please enter root password' },
              { min: 8, message: 'Password must be at least 8 characters' }
            ]}
          >
            <Input.Password placeholder="Minimum 8 characters" />
          </Form.Item>

          <Form.Item
            name="sshKey"
            label="SSH Public Key (Optional)"
            tooltip="Paste your SSH public key for password-less login"
          >
            <Input.TextArea
              rows={3}
              placeholder="ssh-rsa AAAAB3NzaC1yc2EA... user@host"
              style={{ fontFamily: 'monospace', fontSize: 11 }}
            />
          </Form.Item>

          <Form.Item
            name="partitions"
            label="Partition Layout (Optional)"
            tooltip="Leave empty for automatic partitioning"
          >
            <Input placeholder="e.g., /=20G,/home=50G,swap=8G" />
          </Form.Item>

          <Form.Item
            name="packages"
            label="Additional Packages (Optional)"
          >
            <Input placeholder="e.g., nginx,mysql-server,docker.io (comma-separated)" />
          </Form.Item>

          <div style={{ padding: 12, background: '#e6f7ff', border: '1px solid #91d5ff', borderRadius: 4, marginTop: 8 }}>
            <p style={{ margin: 0, fontSize: 12, color: '#0050b3' }}>
              <strong>Process:</strong> PXE boot → IPMI trigger → Power cycle → Install (10-30 min) → Auto reboot
            </p>
          </div>
        </Form>
      </Modal>
      </div>
    </div>
  )
}
