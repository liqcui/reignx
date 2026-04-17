import { useState } from 'react'
import { Card, Table, Tag, Button, Space, Input, Select, Badge, Tabs } from 'antd'
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  SyncOutlined,
  SearchOutlined,
  ReloadOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'

interface Job {
  id: string
  name: string
  type: string
  status: 'running' | 'completed' | 'failed' | 'pending' | 'paused'
  progress: number
  servers: number
  createdAt: string
  updatedAt: string
  user: string
}

export default function Jobs() {
  const [searchText, setSearchText] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('all')

  // Mock data - 模拟大量数据
  const generateMockJobs = (count: number): Job[] => {
    const types = ['patch', 'package', 'os-install', 'config', 'script']
    const statuses: Job['status'][] = ['running', 'completed', 'failed', 'pending', 'paused']
    const jobs: Job[] = []

    for (let i = 1; i <= count; i++) {
      const status = statuses[Math.floor(Math.random() * statuses.length)]
      jobs.push({
        id: `job-${String(i).padStart(4, '0')}`,
        name: `Job ${i} - ${types[i % types.length]}`,
        type: types[i % types.length],
        status,
        progress: status === 'completed' ? 100 : status === 'failed' ? Math.floor(Math.random() * 50) : Math.floor(Math.random() * 90),
        servers: Math.floor(Math.random() * 200) + 1,
        createdAt: new Date(Date.now() - Math.random() * 7 * 24 * 60 * 60 * 1000).toISOString(),
        updatedAt: new Date(Date.now() - Math.random() * 24 * 60 * 60 * 1000).toISOString(),
        user: ['admin', 'operator1', 'operator2'][Math.floor(Math.random() * 3)]
      })
    }
    return jobs
  }

  const [jobs] = useState<Job[]>(generateMockJobs(500))

  const columns: ColumnsType<Job> = [
    {
      title: 'Job ID',
      dataIndex: 'id',
      key: 'id',
      width: 120,
      fixed: 'left',
      sorter: (a, b) => a.id.localeCompare(b.id),
    },
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      width: 200,
      ellipsis: true,
    },
    {
      title: 'Type',
      dataIndex: 'type',
      key: 'type',
      width: 120,
      render: (type: string) => {
        const colorMap: Record<string, string> = {
          patch: 'blue',
          package: 'green',
          'os-install': 'purple',
          config: 'orange',
          script: 'cyan'
        }
        return <Tag color={colorMap[type]}>{type.toUpperCase()}</Tag>
      },
      filters: [
        { text: 'Patch', value: 'patch' },
        { text: 'Package', value: 'package' },
        { text: 'OS Install', value: 'os-install' },
        { text: 'Config', value: 'config' },
        { text: 'Script', value: 'script' },
      ],
      onFilter: (value, record) => record.type === value,
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status: string) => {
        const config = {
          completed: { color: 'green', icon: <CheckCircleOutlined /> },
          running: { color: 'blue', icon: <SyncOutlined spin /> },
          failed: { color: 'red', icon: <CloseCircleOutlined /> },
          pending: { color: 'orange', icon: <PlayCircleOutlined /> },
          paused: { color: 'default', icon: <PauseCircleOutlined /> }
        }[status] || { color: 'default', icon: null }

        return (
          <Tag color={config.color} icon={config.icon}>
            {status.toUpperCase()}
          </Tag>
        )
      },
      filters: [
        { text: 'Running', value: 'running' },
        { text: 'Completed', value: 'completed' },
        { text: 'Failed', value: 'failed' },
        { text: 'Pending', value: 'pending' },
        { text: 'Paused', value: 'paused' },
      ],
      onFilter: (value, record) => record.status === value,
    },
    {
      title: 'Progress',
      dataIndex: 'progress',
      key: 'progress',
      width: 120,
      render: (progress: number) => `${progress}%`,
      sorter: (a, b) => a.progress - b.progress,
    },
    {
      title: 'Servers',
      dataIndex: 'servers',
      key: 'servers',
      width: 100,
      sorter: (a, b) => a.servers - b.servers,
    },
    {
      title: 'User',
      dataIndex: 'user',
      key: 'user',
      width: 120,
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (date: string) => new Date(date).toLocaleString(),
      sorter: (a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 150,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          <Button size="small" type="link">View</Button>
          {record.status === 'running' && <Button size="small" type="link" danger>Stop</Button>}
          {record.status === 'paused' && <Button size="small" type="link">Resume</Button>}
        </Space>
      ),
    },
  ]

  // 过滤数据
  const filteredJobs = jobs.filter(job => {
    const matchSearch = job.name.toLowerCase().includes(searchText.toLowerCase()) ||
                       job.id.toLowerCase().includes(searchText.toLowerCase())
    const matchStatus = statusFilter === 'all' || job.status === statusFilter
    return matchSearch && matchStatus
  })

  // 统计数据
  const stats = {
    total: jobs.length,
    running: jobs.filter(j => j.status === 'running').length,
    completed: jobs.filter(j => j.status === 'completed').length,
    failed: jobs.filter(j => j.status === 'failed').length,
    pending: jobs.filter(j => j.status === 'pending').length,
  }

  // 按类型分组
  const jobsByType = {
    patch: jobs.filter(j => j.type === 'patch'),
    package: jobs.filter(j => j.type === 'package'),
    'os-install': jobs.filter(j => j.type === 'os-install'),
    config: jobs.filter(j => j.type === 'config'),
    script: jobs.filter(j => j.type === 'script'),
  }

  return (
    <div style={{
      padding: '24px',
      minHeight: '100vh',
      background: 'linear-gradient(135deg, #f5f7fa 0%, #e8edf2 100%)'
    }}>
      <div style={{ maxWidth: 1600, margin: '0 auto' }}>
        <div style={{ marginBottom: 20 }}>
          <h2 style={{
            margin: 0,
            fontSize: 21,
            fontWeight: 700,
            background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
            WebkitBackgroundClip: 'text',
            WebkitTextFillColor: 'transparent',
            backgroundClip: 'text'
          }}>
            Jobs Management
          </h2>
          <p style={{ margin: '8px 0 0', color: '#6b7280', fontSize: 13 }}>
            Manage and monitor all job executions
          </p>
        </div>

        {/* Statistics Cards */}
        <div style={{ display: 'flex', gap: 10, marginBottom: 20, flexWrap: 'wrap' }}>
          <Card size="small" style={{ flex: 1, minWidth: 140, borderRadius: 8, boxShadow: '0 2px 8px rgba(0,0,0,0.08)' }}>
            <div style={{ textAlign: 'center' }}>
              <div style={{ color: '#6b7280', fontSize: 12, marginBottom: 4 }}>Total</div>
              <div style={{ fontSize: 22, fontWeight: 700, color: '#667eea' }}>{stats.total}</div>
            </div>
          </Card>
          <Card size="small" style={{ flex: 1, minWidth: 140, borderRadius: 8, boxShadow: '0 2px 8px rgba(0,0,0,0.08)' }}>
            <div style={{ textAlign: 'center' }}>
              <div style={{ color: '#6b7280', fontSize: 12, marginBottom: 4 }}>Running</div>
              <div style={{ fontSize: 22, fontWeight: 700, color: '#1890ff' }}>
                <Badge status="processing" />{stats.running}
              </div>
            </div>
          </Card>
          <Card size="small" style={{ flex: 1, minWidth: 140, borderRadius: 8, boxShadow: '0 2px 8px rgba(0,0,0,0.08)' }}>
            <div style={{ textAlign: 'center' }}>
              <div style={{ color: '#6b7280', fontSize: 12, marginBottom: 4 }}>Completed</div>
              <div style={{ fontSize: 22, fontWeight: 700, color: '#52c41a' }}>{stats.completed}</div>
            </div>
          </Card>
          <Card size="small" style={{ flex: 1, minWidth: 140, borderRadius: 8, boxShadow: '0 2px 8px rgba(0,0,0,0.08)' }}>
            <div style={{ textAlign: 'center' }}>
              <div style={{ color: '#6b7280', fontSize: 12, marginBottom: 4 }}>Failed</div>
              <div style={{ fontSize: 22, fontWeight: 700, color: '#ff4d4f' }}>{stats.failed}</div>
            </div>
          </Card>
          <Card size="small" style={{ flex: 1, minWidth: 140, borderRadius: 8, boxShadow: '0 2px 8px rgba(0,0,0,0.08)' }}>
            <div style={{ textAlign: 'center' }}>
              <div style={{ color: '#6b7280', fontSize: 12, marginBottom: 4 }}>Pending</div>
              <div style={{ fontSize: 22, fontWeight: 700, color: '#faad14' }}>{stats.pending}</div>
            </div>
          </Card>
        </div>

        {/* Tabs for Grouping */}
        <Card bordered={false} style={{ borderRadius: 10, boxShadow: '0 2px 8px rgba(0,0,0,0.08)' }}>
          <Tabs
            defaultActiveKey="all"
            tabBarExtraContent={
              <Space size="small">
                <Input
                  placeholder="Search jobs..."
                  prefix={<SearchOutlined />}
                  value={searchText}
                  onChange={(e) => setSearchText(e.target.value)}
                  style={{ width: 180 }}
                  size="small"
                  allowClear
                />
                <Select value={statusFilter} onChange={setStatusFilter} style={{ width: 110 }} size="small">
                  <Select.Option value="all">All Status</Select.Option>
                  <Select.Option value="running">Running</Select.Option>
                  <Select.Option value="completed">Completed</Select.Option>
                  <Select.Option value="failed">Failed</Select.Option>
                  <Select.Option value="pending">Pending</Select.Option>
                </Select>
                <Button icon={<ReloadOutlined />} size="small">Refresh</Button>
              </Space>
            }
            items={[
              {
                key: 'all',
                label: `All (${filteredJobs.length})`,
                children: (
                  <Table
                    columns={columns}
                    dataSource={filteredJobs}
                    rowKey="id"
                    scroll={{ x: 1400 }}
                    pagination={{
                      pageSize: 20,
                      showSizeChanger: true,
                      showTotal: (total) => `Total ${total} jobs`,
                      pageSizeOptions: ['10', '20', '50', '100']
                    }}
                    size="small"
                  />
                ),
              },
              {
                key: 'patch',
                label: `Patches (${jobsByType.patch.length})`,
                children: (
                  <Table
                    columns={columns}
                    dataSource={jobsByType.patch.filter(job => {
                      const matchSearch = job.name.toLowerCase().includes(searchText.toLowerCase()) ||
                                         job.id.toLowerCase().includes(searchText.toLowerCase())
                      const matchStatus = statusFilter === 'all' || job.status === statusFilter
                      return matchSearch && matchStatus
                    })}
                    rowKey="id"
                    scroll={{ x: 1400 }}
                    pagination={{ pageSize: 20, showSizeChanger: true }}
                    size="small"
                  />
                ),
              },
              {
                key: 'package',
                label: `Packages (${jobsByType.package.length})`,
                children: (
                  <Table
                    columns={columns}
                    dataSource={jobsByType.package.filter(job => {
                      const matchSearch = job.name.toLowerCase().includes(searchText.toLowerCase()) ||
                                         job.id.toLowerCase().includes(searchText.toLowerCase())
                      const matchStatus = statusFilter === 'all' || job.status === statusFilter
                      return matchSearch && matchStatus
                    })}
                    rowKey="id"
                    scroll={{ x: 1400 }}
                    pagination={{ pageSize: 20, showSizeChanger: true }}
                    size="small"
                  />
                ),
              },
              {
                key: 'os-install',
                label: `OS Install (${jobsByType['os-install'].length})`,
                children: (
                  <Table
                    columns={columns}
                    dataSource={jobsByType['os-install'].filter(job => {
                      const matchSearch = job.name.toLowerCase().includes(searchText.toLowerCase()) ||
                                         job.id.toLowerCase().includes(searchText.toLowerCase())
                      const matchStatus = statusFilter === 'all' || job.status === statusFilter
                      return matchSearch && matchStatus
                    })}
                    rowKey="id"
                    scroll={{ x: 1400 }}
                    pagination={{ pageSize: 20, showSizeChanger: true }}
                    size="small"
                  />
                ),
              },
              {
                key: 'config',
                label: `Config (${jobsByType.config.length})`,
                children: (
                  <Table
                    columns={columns}
                    dataSource={jobsByType.config.filter(job => {
                      const matchSearch = job.name.toLowerCase().includes(searchText.toLowerCase()) ||
                                         job.id.toLowerCase().includes(searchText.toLowerCase())
                      const matchStatus = statusFilter === 'all' || job.status === statusFilter
                      return matchSearch && matchStatus
                    })}
                    rowKey="id"
                    scroll={{ x: 1400 }}
                    pagination={{ pageSize: 20, showSizeChanger: true }}
                    size="small"
                  />
                ),
              },
              {
                key: 'script',
                label: `Scripts (${jobsByType.script.length})`,
                children: (
                  <Table
                    columns={columns}
                    dataSource={jobsByType.script.filter(job => {
                      const matchSearch = job.name.toLowerCase().includes(searchText.toLowerCase()) ||
                                         job.id.toLowerCase().includes(searchText.toLowerCase())
                      const matchStatus = statusFilter === 'all' || job.status === statusFilter
                      return matchSearch && matchStatus
                    })}
                    rowKey="id"
                    scroll={{ x: 1400 }}
                    pagination={{ pageSize: 20, showSizeChanger: true }}
                    size="small"
                  />
                ),
              },
            ]}
          />
        </Card>
      </div>
    </div>
  )
}
