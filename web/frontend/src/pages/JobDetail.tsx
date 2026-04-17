import { useParams } from 'react-router-dom'
import { Card, Descriptions, Tag, Table, Progress, Space, Button, Timeline } from 'antd'
import { useState, useEffect } from 'react'
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  SyncOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'

interface Task {
  id: string
  serverId: string
  hostname: string
  status: string
  attempts: number
  exitCode: number | null
  output: string
  startedAt: string
  completedAt: string | null
}

interface JobDetail {
  id: string
  name: string
  type: string
  status: string
  priority: string
  progress: number
  targetServers: number
  completedTasks: number
  failedTasks: number
  pendingTasks: number
  createdAt: string
  startedAt: string | null
  completedAt: string | null
  createdBy: string
  parameters: Record<string, any>
}

export default function JobDetail() {
  const { id } = useParams<{ id: string }>()
  const [job, setJob] = useState<JobDetail | null>(null)
  const [tasks, setTasks] = useState<Task[]>([])

  useEffect(() => {
    // TODO: Replace with actual API call
    setJob({
      id: id || '',
      name: 'Security Patches - April 2026',
      type: 'patch',
      status: 'running',
      priority: 'high',
      progress: 75,
      targetServers: 50,
      completedTasks: 38,
      failedTasks: 0,
      pendingTasks: 12,
      createdAt: '2026-04-15 10:30:00',
      startedAt: '2026-04-15 10:31:00',
      completedAt: null,
      createdBy: 'admin',
      parameters: {
        patch_ids: ['CVE-2026-1234', 'CVE-2026-5678'],
        reboot_if_required: true,
      },
    })

    setTasks([
      {
        id: 'task-001',
        serverId: 'server-001',
        hostname: 'web-01.example.com',
        status: 'completed',
        attempts: 1,
        exitCode: 0,
        output: 'Successfully installed 5 patches',
        startedAt: '2026-04-15 10:31:05',
        completedAt: '2026-04-15 10:35:20',
      },
      {
        id: 'task-002',
        serverId: 'server-002',
        hostname: 'db-01.example.com',
        status: 'running',
        attempts: 1,
        exitCode: null,
        output: 'Installing patches...',
        startedAt: '2026-04-15 10:31:10',
        completedAt: null,
      },
      {
        id: 'task-003',
        serverId: 'server-003',
        hostname: 'app-01.example.com',
        status: 'pending',
        attempts: 0,
        exitCode: null,
        output: '',
        startedAt: '',
        completedAt: null,
      },
    ])
  }, [id])

  if (!job) {
    return <div>Loading...</div>
  }

  const statusColor =
    job.status === 'completed'
      ? 'green'
      : job.status === 'running'
      ? 'blue'
      : job.status === 'failed'
      ? 'red'
      : 'orange'

  const taskColumns: ColumnsType<Task> = [
    {
      title: 'Task ID',
      dataIndex: 'id',
      key: 'id',
    },
    {
      title: 'Server',
      dataIndex: 'hostname',
      key: 'hostname',
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
    },
    {
      title: 'Exit Code',
      dataIndex: 'exitCode',
      key: 'exitCode',
      render: (code: number | null) => (code !== null ? code : '-'),
    },
    {
      title: 'Started At',
      dataIndex: 'startedAt',
      key: 'startedAt',
      render: (time: string) => time || '-',
    },
    {
      title: 'Completed At',
      dataIndex: 'completedAt',
      key: 'completedAt',
      render: (time: string | null) => time || '-',
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

  return (
    <div>
      <Card style={{ marginBottom: 16 }}>
        <div style={{ marginBottom: 24 }}>
          <Space style={{ float: 'right' }}>
            {job.status === 'running' && (
              <Button danger>Cancel Job</Button>
            )}
            {job.status === 'failed' && (
              <Button type="primary">Retry Failed Tasks</Button>
            )}
          </Space>
          <h2>{job.name}</h2>
          <Space>
            <Tag color={statusColor}>{job.status.toUpperCase()}</Tag>
            <Tag color="blue">{job.type.toUpperCase()}</Tag>
            <Tag color="purple">Priority: {job.priority.toUpperCase()}</Tag>
          </Space>
        </div>

        <div style={{ marginBottom: 24 }}>
          <h3>Progress</h3>
          <Progress
            percent={job.progress}
            status={job.status === 'failed' ? 'exception' : job.status === 'completed' ? 'success' : 'active'}
            format={(percent) => `${percent}% (${job.completedTasks}/${job.targetServers})`}
          />
        </div>

        <Descriptions bordered column={2}>
          <Descriptions.Item label="Job ID">{job.id}</Descriptions.Item>
          <Descriptions.Item label="Type">{job.type}</Descriptions.Item>
          <Descriptions.Item label="Priority">{job.priority}</Descriptions.Item>
          <Descriptions.Item label="Created By">{job.createdBy}</Descriptions.Item>
          <Descriptions.Item label="Target Servers">{job.targetServers}</Descriptions.Item>
          <Descriptions.Item label="Completed Tasks">
            <Tag color="green">{job.completedTasks}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="Pending Tasks">
            <Tag color="orange">{job.pendingTasks}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="Failed Tasks">
            <Tag color="red">{job.failedTasks}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="Created At">{job.createdAt}</Descriptions.Item>
          <Descriptions.Item label="Started At">{job.startedAt || 'Not started'}</Descriptions.Item>
          <Descriptions.Item label="Completed At" span={2}>
            {job.completedAt || 'In progress'}
          </Descriptions.Item>
          <Descriptions.Item label="Parameters" span={2}>
            <pre style={{ margin: 0 }}>{JSON.stringify(job.parameters, null, 2)}</pre>
          </Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title="Task Details">
        <Table
          columns={taskColumns}
          dataSource={tasks}
          rowKey="id"
          pagination={{
            pageSize: 20,
            showTotal: (total) => `Total ${total} tasks`,
          }}
          expandable={{
            expandedRowRender: (record) => (
              <div>
                <h4>Output:</h4>
                <pre
                  style={{
                    background: '#f5f5f5',
                    padding: 16,
                    borderRadius: 4,
                    maxHeight: 300,
                    overflow: 'auto',
                  }}
                >
                  {record.output || 'No output available'}
                </pre>
              </div>
            ),
          }}
        />
      </Card>

      <Card title="Job Timeline" style={{ marginTop: 16 }}>
        <Timeline
          items={[
            {
              color: 'green',
              children: (
                <div>
                  <strong>Job Created</strong>
                  <br />
                  {job.createdAt} by {job.createdBy}
                </div>
              ),
            },
            {
              color: job.startedAt ? 'green' : 'gray',
              children: (
                <div>
                  <strong>Job Started</strong>
                  <br />
                  {job.startedAt || 'Not started yet'}
                </div>
              ),
            },
            {
              color: job.status === 'running' ? 'blue' : job.completedAt ? 'green' : 'gray',
              dot: job.status === 'running' ? <SyncOutlined spin /> : undefined,
              children: (
                <div>
                  <strong>
                    {job.status === 'running'
                      ? 'In Progress'
                      : job.status === 'completed'
                      ? 'Completed'
                      : job.status === 'failed'
                      ? 'Failed'
                      : 'Pending'}
                  </strong>
                  <br />
                  {job.completedTasks} of {job.targetServers} tasks completed
                  {job.failedTasks > 0 && `, ${job.failedTasks} failed`}
                </div>
              ),
            },
          ]}
        />
      </Card>
    </div>
  )
}
