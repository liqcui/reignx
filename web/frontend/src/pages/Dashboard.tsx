import { Row, Col, Card, Table, Tag } from 'antd'
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  SyncOutlined,
  CloudServerOutlined,
} from '@ant-design/icons'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'
import { useState } from 'react'

export default function Dashboard() {
  const [stats] = useState({
    totalServers: 156,
    activeServers: 142,
    failedServers: 8,
    pendingJobs: 12,
  })

  const [recentJobs] = useState([
    { id: 'job-001', name: 'Security Patches', status: 'running', progress: 75, servers: 50 },
    { id: 'job-002', name: 'Package Updates', status: 'completed', progress: 100, servers: 120 },
    { id: 'job-003', name: 'Config Deployment', status: 'pending', progress: 0, servers: 30 },
  ])

  const [metricsData] = useState([
    { time: '00:00', tasks: 120, success: 115, failed: 5 },
    { time: '04:00', tasks: 98, success: 95, failed: 3 },
    { time: '08:00', tasks: 156, success: 150, failed: 6 },
    { time: '12:00', tasks: 201, success: 195, failed: 6 },
    { time: '16:00', tasks: 178, success: 172, failed: 6 },
    { time: '20:00', tasks: 145, success: 140, failed: 5 },
  ])

  const columns = [
    {
      title: 'Job ID',
      dataIndex: 'id',
      key: 'id',
    },
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const color = status === 'completed' ? 'green' : status === 'running' ? 'blue' : 'orange'
        const icon = status === 'completed' ? <CheckCircleOutlined /> :
                     status === 'running' ? <SyncOutlined spin /> : null
        return <Tag color={color} icon={icon}>{status.toUpperCase()}</Tag>
      },
    },
    {
      title: 'Progress',
      dataIndex: 'progress',
      key: 'progress',
      render: (progress: number) => `${progress}%`,
    },
    {
      title: 'Servers',
      dataIndex: 'servers',
      key: 'servers',
    },
  ]

  return (
    <div style={{
      padding: '16px',
      minHeight: '100vh',
      background: '#f5f7fa'
    }}>
      <div style={{ maxWidth: 1600, margin: '0 auto' }}>
        <div style={{ marginBottom: 16 }}>
          <h2 style={{
            margin: 0,
            fontSize: 21,
            fontWeight: 600,
            color: '#1f2937'
          }}>
            Dashboard
          </h2>
          <p style={{ margin: '4px 0 0', color: '#6b7280', fontSize: 13 }}>
            Overview of your server infrastructure
          </p>
        </div>

        <Row gutter={[10, 10]}>
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
              bodyStyle={{ padding: '10px 14px' }}
              onMouseEnter={(e) => {
                e.currentTarget.style.transform = 'translateY(-1px)'
                e.currentTarget.style.boxShadow = '0 3px 10px rgba(102, 126, 234, 0.25)'
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.transform = 'translateY(0)'
                e.currentTarget.style.boxShadow = '0 2px 8px rgba(102, 126, 234, 0.2)'
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <div style={{
                  background: 'rgba(255, 255, 255, 0.15)',
                  width: 38,
                  height: 38,
                  borderRadius: 8,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  flexShrink: 0,
                }}>
                  <CloudServerOutlined style={{ fontSize: 18, color: '#ffffff' }} />
                </div>
                <div style={{ flex: 1 }}>
                  <div style={{ color: 'rgba(255, 255, 255, 0.85)', fontSize: 11, fontWeight: 500, marginBottom: 2 }}>
                    Total Servers
                  </div>
                  <div style={{ color: '#ffffff', fontSize: 20, fontWeight: 700, lineHeight: 1 }}>
                    {stats.totalServers}
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
              bodyStyle={{ padding: '10px 14px' }}
              onMouseEnter={(e) => {
                e.currentTarget.style.transform = 'translateY(-1px)'
                e.currentTarget.style.boxShadow = '0 3px 10px rgba(82, 196, 26, 0.25)'
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.transform = 'translateY(0)'
                e.currentTarget.style.boxShadow = '0 2px 8px rgba(82, 196, 26, 0.2)'
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <div style={{
                  background: 'rgba(255, 255, 255, 0.15)',
                  width: 38,
                  height: 38,
                  borderRadius: 8,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  flexShrink: 0,
                }}>
                  <CheckCircleOutlined style={{ fontSize: 18, color: '#ffffff' }} />
                </div>
                <div style={{ flex: 1 }}>
                  <div style={{ color: 'rgba(255, 255, 255, 0.85)', fontSize: 11, fontWeight: 500, marginBottom: 2 }}>
                    Active Servers
                  </div>
                  <div style={{ color: '#ffffff', fontSize: 20, fontWeight: 700, lineHeight: 1 }}>
                    {stats.activeServers}
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
                background: 'linear-gradient(135deg, #ff4d4f 0%, #cf1322 100%)',
                boxShadow: '0 2px 8px rgba(255, 77, 79, 0.2)',
                transition: 'all 0.2s ease',
                cursor: 'pointer',
              }}
              bodyStyle={{ padding: '10px 14px' }}
              onMouseEnter={(e) => {
                e.currentTarget.style.transform = 'translateY(-1px)'
                e.currentTarget.style.boxShadow = '0 3px 10px rgba(255, 77, 79, 0.25)'
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.transform = 'translateY(0)'
                e.currentTarget.style.boxShadow = '0 2px 8px rgba(255, 77, 79, 0.2)'
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <div style={{
                  background: 'rgba(255, 255, 255, 0.15)',
                  width: 38,
                  height: 38,
                  borderRadius: 8,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  flexShrink: 0,
                }}>
                  <CloseCircleOutlined style={{ fontSize: 18, color: '#ffffff' }} />
                </div>
                <div style={{ flex: 1 }}>
                  <div style={{ color: 'rgba(255, 255, 255, 0.85)', fontSize: 11, fontWeight: 500, marginBottom: 2 }}>
                    Failed Servers
                  </div>
                  <div style={{ color: '#ffffff', fontSize: 20, fontWeight: 700, lineHeight: 1 }}>
                    {stats.failedServers}
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
              bodyStyle={{ padding: '10px 14px' }}
              onMouseEnter={(e) => {
                e.currentTarget.style.transform = 'translateY(-1px)'
                e.currentTarget.style.boxShadow = '0 3px 10px rgba(250, 173, 20, 0.25)'
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.transform = 'translateY(0)'
                e.currentTarget.style.boxShadow = '0 2px 8px rgba(250, 173, 20, 0.2)'
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <div style={{
                  background: 'rgba(255, 255, 255, 0.15)',
                  width: 38,
                  height: 38,
                  borderRadius: 8,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  flexShrink: 0,
                }}>
                  <SyncOutlined style={{ fontSize: 18, color: '#ffffff' }} />
                </div>
                <div style={{ flex: 1 }}>
                  <div style={{ color: 'rgba(255, 255, 255, 0.85)', fontSize: 11, fontWeight: 500, marginBottom: 2 }}>
                    Pending Jobs
                  </div>
                  <div style={{ color: '#ffffff', fontSize: 20, fontWeight: 700, lineHeight: 1 }}>
                    {stats.pendingJobs}
                  </div>
                </div>
              </div>
            </Card>
          </Col>
        </Row>

        <Row gutter={[12, 12]} style={{ marginTop: 12 }}>
          <Col xs={24} lg={16}>
            <Card
              bordered={false}
              title={
                <span style={{ fontSize: 14, fontWeight: 600, color: '#1f2937' }}>
                  Task Execution Metrics
                </span>
              }
              style={{
                borderRadius: 8,
                boxShadow: '0 1px 4px rgba(0,0,0,0.06)'
              }}
              bodyStyle={{ padding: '16px' }}
            >
              <ResponsiveContainer width="100%" height={240}>
                <LineChart data={metricsData}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                  <XAxis dataKey="time" stroke="#6b7280" />
                  <YAxis stroke="#6b7280" />
                  <Tooltip
                    contentStyle={{
                      borderRadius: 8,
                      border: 'none',
                      boxShadow: '0 4px 12px rgba(0,0,0,0.1)'
                    }}
                  />
                  <Legend />
                  <Line
                    type="monotone"
                    dataKey="tasks"
                    stroke="#667eea"
                    strokeWidth={2}
                    name="Total Tasks"
                    dot={{ fill: '#667eea', r: 4 }}
                  />
                  <Line
                    type="monotone"
                    dataKey="success"
                    stroke="#52c41a"
                    strokeWidth={2}
                    name="Success"
                    dot={{ fill: '#52c41a', r: 4 }}
                  />
                  <Line
                    type="monotone"
                    dataKey="failed"
                    stroke="#ff4d4f"
                    strokeWidth={2}
                    name="Failed"
                    dot={{ fill: '#ff4d4f', r: 4 }}
                  />
                </LineChart>
              </ResponsiveContainer>
            </Card>
          </Col>
          <Col xs={24} lg={8}>
            <Card
              bordered={false}
              title={
                <span style={{ fontSize: 14, fontWeight: 600, color: '#1f2937' }}>
                  System Health
                </span>
              }
              style={{
                borderRadius: 8,
                boxShadow: '0 1px 4px rgba(0,0,0,0.06)'
              }}
              bodyStyle={{ padding: '16px' }}
            >
              <div style={{ marginBottom: 16 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6 }}>
                  <span style={{ color: '#6b7280', fontSize: 12 }}>CPU Usage</span>
                  <span style={{ fontWeight: 600, color: '#52c41a', fontSize: 12 }}>45%</span>
                </div>
                <div style={{
                  background: '#e5e7eb',
                  height: 8,
                  borderRadius: 4,
                  overflow: 'hidden'
                }}>
                  <div style={{
                    background: 'linear-gradient(90deg, #52c41a 0%, #73d13d 100%)',
                    width: '45%',
                    height: '100%',
                    transition: 'width 0.3s ease'
                  }} />
                </div>
              </div>
              <div style={{ marginBottom: 16 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6 }}>
                  <span style={{ color: '#6b7280', fontSize: 12 }}>Memory Usage</span>
                  <span style={{ fontWeight: 600, color: '#1890ff', fontSize: 12 }}>68%</span>
                </div>
                <div style={{
                  background: '#e5e7eb',
                  height: 8,
                  borderRadius: 4,
                  overflow: 'hidden'
                }}>
                  <div style={{
                    background: 'linear-gradient(90deg, #1890ff 0%, #40a9ff 100%)',
                    width: '68%',
                    height: '100%',
                    transition: 'width 0.3s ease'
                  }} />
                </div>
              </div>
              <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6 }}>
                  <span style={{ color: '#6b7280', fontSize: 12 }}>Disk Usage</span>
                  <span style={{ fontWeight: 600, color: '#faad14', fontSize: 12 }}>82%</span>
                </div>
                <div style={{
                  background: '#e5e7eb',
                  height: 8,
                  borderRadius: 4,
                  overflow: 'hidden'
                }}>
                  <div style={{
                    background: 'linear-gradient(90deg, #faad14 0%, #ffc53d 100%)',
                    width: '82%',
                    height: '100%',
                    transition: 'width 0.3s ease'
                  }} />
                </div>
              </div>
            </Card>
          </Col>
        </Row>

        <Row gutter={[12, 12]} style={{ marginTop: 12 }}>
          <Col xs={24}>
            <Card
              bordered={false}
              title={
                <span style={{ fontSize: 14, fontWeight: 600, color: '#1f2937' }}>
                  Recent Jobs
                </span>
              }
              extra={
                <a
                  href="/jobs"
                  style={{
                    color: '#667eea',
                    fontWeight: 500,
                    fontSize: 13,
                    textDecoration: 'none'
                  }}
                >
                  View All →
                </a>
              }
              style={{
                borderRadius: 8,
                boxShadow: '0 1px 4px rgba(0,0,0,0.06)'
              }}
              bodyStyle={{ padding: '16px' }}
            >
              <Table
                columns={columns}
                dataSource={recentJobs}
                rowKey="id"
                pagination={false}
                size="middle"
              />
            </Card>
          </Col>
        </Row>
      </div>
    </div>
  )
}
