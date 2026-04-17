import { useState } from 'react'
import { Table, Tag, Space, Input, Select, Card, Button } from 'antd'
import { SearchOutlined, ReloadOutlined, CheckCircleOutlined, CloseCircleOutlined, SyncOutlined, ClockCircleOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import type { ColumnsType } from 'antd/es/table'

interface Task {
  id: string
  jobId: string
  jobName: string
  serverId: string
  hostname: string
  taskType: string
  status: string
  attempts: number
  exitCode: number | null
  startedAt: string
  completedAt: string | null
  duration: string | null
}

export default function Tasks() {
  const navigate = useNavigate()
  const [searchText, setSearchText] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [typeFilter, setTypeFilter] = useState<string>('all')

  const [tasks] = useState<Task[]>([
    {
      id: 'task-001',
      jobId: 'job-001',
      jobName: 'Security Patches - April 2026',
      serverId: 'server-001',
      hostname: 'web-01.example.com',
      taskType: 'patch',
      status: 'completed',
      attempts: 1,
      exitCode: 0,
      startedAt: '2026-04-15 10:31:05',
      completedAt: '2026-04-15 10:35:20',
      duration: '4m 15s',
    },
    {
      id: 'task-002',
      jobId: 'job-001',
      jobName: 'Security Patches - April 2026',
      serverId: 'server-002',
      hostname: 'db-01.example.com',
      taskType: 'patch',
      status: 'running',
      attempts: 1,
      exitCode: null,
      startedAt: '2026-04-15 10:31:10',
      completedAt: null,
      duration: null,
    },
    {
      id: 'task-003',
      jobId: 'job-001',
      jobName: 'Security Patches - April 2026',
      serverId: 'server-003',
      hostname: 'app-01.example.com',
      taskType: 'patch',
      status: 'pending',
      attempts: 0,
      exitCode: null,
      startedAt: '',
      completedAt: null,
      duration: null,
    },
    {
      id: 'task-004',
      jobId: 'job-002',
      jobName: 'Package Updates - Web Servers',
      serverId: 'server-004',
      hostname: 'web-02.example.com',
      taskType: 'package',
      status: 'completed',
      attempts: 1,
      exitCode: 0,
      startedAt: '2026-04-14 08:16:00',
      completedAt: '2026-04-14 08:18:30',
      duration: '2m 30s',
    },
    {
      id: 'task-005',
      jobId: 'job-004',
      jobName: 'OS Upgrade - Ubuntu 24.04',
      serverId: 'server-010',
      hostname: 'test-server-01.example.com',
      taskType: 'upgrade',
      status: 'failed',
      attempts: 3,
      exitCode: 1,
      startedAt: '2026-04-13 16:05:00',
      completedAt: '2026-04-13 16:25:00',
      duration: '20m 0s',
    },
  ])

  const columns: ColumnsType<Task> = [
    {
      title: 'Task ID',
      dataIndex: 'id',
      key: 'id',
    },
    {
      title: 'Job',
      key: 'job',
      render: (_, record) => (
        <Button type="link" onClick={() => navigate(`/jobs/${record.jobId}`)}>
          {record.jobName}
        </Button>
      ),
    },
    {
      title: 'Server',
      dataIndex: 'hostname',
      key: 'hostname',
      render: (hostname: string, record) => (
        <Button type="link" onClick={() => navigate(`/servers/${record.serverId}`)}>
          {hostname}
        </Button>
      ),
    },
    {
      title: 'Type',
      dataIndex: 'taskType',
      key: 'taskType',
      render: (type: string) => {
        const colors: Record<string, string> = {
          patch: 'blue',
          package: 'green',
          deploy: 'purple',
          upgrade: 'orange',
          install_os: 'cyan',
        }
        return <Tag color={colors[type] || 'default'}>{type.toUpperCase()}</Tag>
      },
      filters: [
        { text: 'Patch', value: 'patch' },
        { text: 'Package', value: 'package' },
        { text: 'Deploy', value: 'deploy' },
        { text: 'Upgrade', value: 'upgrade' },
        { text: 'Install OS', value: 'install_os' },
      ],
      onFilter: (value, record) => record.taskType === value,
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const color =
          status === 'completed'
            ? 'green'
            : status === 'running'
            ? 'blue'
            : status === 'failed'
            ? 'red'
            : 'orange'
        const icon =
          status === 'completed' ? (
            <CheckCircleOutlined />
          ) : status === 'running' ? (
            <SyncOutlined spin />
          ) : status === 'failed' ? (
            <CloseCircleOutlined />
          ) : (
            <ClockCircleOutlined />
          )
        return (
          <Tag color={color} icon={icon}>
            {status.toUpperCase()}
          </Tag>
        )
      },
      filters: [
        { text: 'Pending', value: 'pending' },
        { text: 'Running', value: 'running' },
        { text: 'Completed', value: 'completed' },
        { text: 'Failed', value: 'failed' },
      ],
      onFilter: (value, record) => record.status === value,
    },
    {
      title: 'Attempts',
      dataIndex: 'attempts',
      key: 'attempts',
      render: (attempts: number) => (
        <Tag color={attempts > 1 ? 'orange' : 'default'}>{attempts}</Tag>
      ),
    },
    {
      title: 'Exit Code',
      dataIndex: 'exitCode',
      key: 'exitCode',
      render: (code: number | null) => {
        if (code === null) return '-'
        return (
          <Tag color={code === 0 ? 'green' : 'red'}>{code}</Tag>
        )
      },
    },
    {
      title: 'Duration',
      dataIndex: 'duration',
      key: 'duration',
      render: (duration: string | null) => duration || '-',
    },
    {
      title: 'Started At',
      dataIndex: 'startedAt',
      key: 'startedAt',
      render: (time: string) => time || '-',
      sorter: (a, b) => {
        if (!a.startedAt) return 1
        if (!b.startedAt) return -1
        return new Date(a.startedAt).getTime() - new Date(b.startedAt).getTime()
      },
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_, record) => (
        <Space>
          <Button type="link" size="small">
            View Output
          </Button>
          {record.status === 'failed' && (
            <Button type="link" size="small">
              Retry
            </Button>
          )}
        </Space>
      ),
    },
  ]

  const filteredTasks = tasks.filter((task) => {
    const matchesSearch =
      task.hostname.toLowerCase().includes(searchText.toLowerCase()) ||
      task.id.includes(searchText) ||
      task.jobName.toLowerCase().includes(searchText.toLowerCase())
    const matchesStatus = statusFilter === 'all' || task.status === statusFilter
    const matchesType = typeFilter === 'all' || task.taskType === typeFilter
    return matchesSearch && matchesStatus && matchesType
  })

  return (
    <div style={{ width: '100%' }}>
      <div style={{ marginBottom: 24 }}>
        <h1 style={{ margin: 0, fontSize: 21, fontWeight: 600 }}>Tasks</h1>
        <p style={{ margin: '8px 0 0', color: '#666' }}>Monitor task execution across all jobs</p>
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
              placeholder="Search tasks..."
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              style={{ width: 300 }}
            />
            <Select value={statusFilter} onChange={setStatusFilter} style={{ width: 120 }}>
              <Select.Option value="all">All Status</Select.Option>
              <Select.Option value="pending">Pending</Select.Option>
              <Select.Option value="running">Running</Select.Option>
              <Select.Option value="completed">Completed</Select.Option>
              <Select.Option value="failed">Failed</Select.Option>
            </Select>
            <Select value={typeFilter} onChange={setTypeFilter} style={{ width: 120 }}>
              <Select.Option value="all">All Types</Select.Option>
              <Select.Option value="patch">Patch</Select.Option>
              <Select.Option value="package">Package</Select.Option>
              <Select.Option value="deploy">Deploy</Select.Option>
              <Select.Option value="upgrade">Upgrade</Select.Option>
              <Select.Option value="install_os">Install OS</Select.Option>
            </Select>
            <Button icon={<ReloadOutlined />}>Refresh</Button>
          </Space>
        </div>
        <Table
          columns={columns}
          dataSource={filteredTasks}
          rowKey="id"
          scroll={{ x: 1600 }}
          pagination={{
            pageSize: 20,
            showTotal: (total) => `Total ${total} tasks`,
            showSizeChanger: true,
            showQuickJumper: true,
            pageSizeOptions: ['10', '20', '50', '100'],
          }}
        />
      </Card>
    </div>
  )
}
