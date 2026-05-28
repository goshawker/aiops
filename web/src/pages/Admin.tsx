import { Card, Typography } from 'antd'

const { Title, Paragraph } = Typography

export default function Admin() {
  return (
    <div>
      <Title level={4}>管理配置</Title>
      <Card>
        <Paragraph>
          管理功能开发中，包含：
        </Paragraph>
        <ul>
          <li>用户与权限管理（RBAC）</li>
          <li>采集器管理（Agent 列表、配置下发）</li>
          <li>系统配置（数据保留、通知渠道）</li>
          <li>审计日志</li>
        </ul>
      </Card>
    </div>
  )
}
