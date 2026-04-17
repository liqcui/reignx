import { useState, useEffect } from 'react'
import { Table, Button, Tag, Space, Input, Select, Card, Modal, Form, message } from 'antd'
import { PlusOutlined, SearchOutlined, ReloadOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import type { ColumnsType } from 'antd/es/table'
import { serversAPI } from '@/lib/api'

interface Server {
  id: string
  hostname: string
  ip: string
  os: string
  status: string
  mode: string
  last_seen?: string
  group?: string
}

export default function Servers() {
  const navigate = useNavigate()
  const [searchText, setSearchText] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [groupFilter, setGroupFilter] = useState<string>('all')
  const [isAddModalOpen, setIsAddModalOpen] = useState(false)
  const [addForm] = Form.useForm()
  const [servers, setServers] = useState<Server[]>([])
  const [loading, setLoading] = useState(false)

  const fetchServers = async () => {
    setLoading(true)
    try {
      const response = await serversAPI.list()
      setServers(response.servers || [])
    } catch (error: any) {
      message.error('Failed to load servers')
      console.error('Failed to fetch servers:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchServers()
  }, [])

  // Get unique groups for filter
  const groups = Array.from(new Set(servers.map(s => s.group).filter(Boolean))) as string[]

  const columns: ColumnsType<Server> = [
    {
      title: 'Hostname',
      dataIndex: 'hostname',
      key: 'hostname',
      sorter: (a, b) => a.hostname.localeCompare(b.hostname),
    },
    {
      title: 'IP Address',
      dataIndex: 'ip',
      key: 'ip',
    },
    {
      title: 'OS',
      dataIndex: 'os',
      key: 'os',
    },
    {
      title: 'Group',
      dataIndex: 'group',
      key: 'group',
      render: (group: string) => {
        if (!group) return <Tag>Unassigned</Tag>
        const colors: Record<string, string> = {
          dev: 'blue',
          sit: 'cyan',
          uat: 'orange',
          staging: 'purple',
          prod: 'red',
        }
        const labels: Record<string, string> = {
          dev: 'DEV',
          sit: 'SIT',
          uat: 'UAT',
          staging: 'STAGING',
          prod: 'PROD',
        }
        return <Tag color={colors[group] || 'default'}>{labels[group] || group.toUpperCase()}</Tag>
      },
      filters: groups.map(g => ({ text: g.toUpperCase(), value: g })),
      onFilter: (value, record) => record.group === value,
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const color = status === 'online' ? 'green' : status === 'offline' ? 'red' : 'orange'
        return <Tag color={color}>{status.toUpperCase()}</Tag>
      },
      filters: [
        { text: 'Online', value: 'online' },
        { text: 'Offline', value: 'offline' },
      ],
      onFilter: (value, record) => record.status === value,
    },
    {
      title: 'Mode',
      dataIndex: 'mode',
      key: 'mode',
      render: (mode: string) => (
        <Tag color={mode === 'agent' ? 'blue' : 'purple'}>{mode.toUpperCase()}</Tag>
      ),
    },
    {
      title: 'Last Seen',
      dataIndex: 'last_seen',
      key: 'last_seen',
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_, record) => (
        <Space>
          <Button type="link" onClick={() => navigate(`/servers/${record.id}`)}>
            View
          </Button>
          <Button type="link" danger>
            Remove
          </Button>
        </Space>
      ),
    },
  ]

  const filteredServers = servers.filter((server) => {
    const matchesSearch = server.hostname.toLowerCase().includes(searchText.toLowerCase()) ||
                         server.ip.includes(searchText)
    const matchesStatus = statusFilter === 'all' || server.status === statusFilter
    const matchesGroup = groupFilter === 'all' || server.group === groupFilter
    return matchesSearch && matchesStatus && matchesGroup
  })

  const handleAddServer = async (values: any) => {
    try {
      console.log('Adding server:', values)
      // TODO: Replace with actual API call
      message.success(`Server ${values.hostname} added successfully!`)
      setIsAddModalOpen(false)
      addForm.resetFields()
    } catch (error) {
      message.error('Failed to add server')
    }
  }

  return (
    <div style={{ width: '100%' }}>
      <div style={{ marginBottom: 24 }}>
        <h1 style={{ margin: 0, fontSize: 21, fontWeight: 600 }}>Servers</h1>
        <p style={{ margin: '8px 0 0', color: '#666' }}>Manage your server infrastructure</p>
      </div>

      <Card>
        <div style={{
          marginBottom: 16,
          display: 'flex',
          flexWrap: 'wrap',
          gap: 12,
          justifyContent: 'space-between',
          alignItems: 'center',
        }}>
          <Space>
            <Input
              placeholder="Search servers..."
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              style={{ width: 300 }}
            />
            <Select
              value={groupFilter}
              onChange={setGroupFilter}
              style={{ width: 120 }}
            >
              <Select.Option value="all">All Groups</Select.Option>
              {groups.map(group => (
                <Select.Option key={group} value={group}>{group.toUpperCase()}</Select.Option>
              ))}
            </Select>
            <Select
              value={statusFilter}
              onChange={setStatusFilter}
              style={{ width: 120 }}
            >
              <Select.Option value="all">All Status</Select.Option>
              <Select.Option value="online">Online</Select.Option>
              <Select.Option value="offline">Offline</Select.Option>
            </Select>
            <Button icon={<ReloadOutlined />} onClick={fetchServers} loading={loading}>
              Refresh
            </Button>
          </Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setIsAddModalOpen(true)}>
            Add Server
          </Button>
        </div>
        <Table
          columns={columns}
          dataSource={filteredServers}
          rowKey="id"
          loading={loading}
          scroll={{ x: 1200 }}
          pagination={{
            pageSize: 10,
            showTotal: (total) => `Total ${total} servers`,
            showSizeChanger: true,
            showQuickJumper: true,
          }}
        />
      </Card>

      <Modal
        title="Add New Server"
        open={isAddModalOpen}
        onCancel={() => {
          setIsAddModalOpen(false)
          addForm.resetFields()
        }}
        onOk={() => addForm.submit()}
        width={600}
      >
        <Form
          form={addForm}
          layout="vertical"
          onFinish={handleAddServer}
        >
          <Form.Item
            name="hostname"
            label="Hostname"
            rules={[{ required: true, message: 'Please enter hostname' }]}
          >
            <Input placeholder="e.g., web-server-01.example.com" />
          </Form.Item>

          <Form.Item
            name="ip"
            label="IP Address"
            rules={[
              { required: true, message: 'Please enter IP address' },
              { pattern: /^(\d{1,3}\.){3}\d{1,3}$/, message: 'Please enter valid IP address' }
            ]}
          >
            <Input placeholder="e.g., 192.168.1.100" />
          </Form.Item>

          <Form.Item
            name="os"
            label="Operating System"
            rules={[{ required: true, message: 'Please select OS' }]}
          >
            <Select placeholder="Select operating system">
              <Select.Option value="Ubuntu 22.04">Ubuntu 22.04</Select.Option>
              <Select.Option value="Ubuntu 20.04">Ubuntu 20.04</Select.Option>
              <Select.Option value="CentOS 7">CentOS 7</Select.Option>
              <Select.Option value="RHEL 8">RHEL 8</Select.Option>
              <Select.Option value="Debian 11">Debian 11</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="mode"
            label="Connection Mode"
            rules={[{ required: true, message: 'Please select mode' }]}
            initialValue="agent"
          >
            <Select>
              <Select.Option value="agent">Agent (Recommended)</Select.Option>
              <Select.Option value="ssh">SSH (Agentless)</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="group"
            label="Server Group"
            rules={[{ required: true, message: 'Please select or enter a group' }]}
          >
            <Select
              placeholder="Select or enter group name"
              mode="tags"
              maxCount={1}
              options={groups.map(g => ({ label: g.toUpperCase(), value: g }))}
            />
          </Form.Item>

          <Form.Item
            name="tags"
            label="Tags (Optional)"
          >
            <Input placeholder="e.g., production, web-server" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
