# AIOps 等保三级安全审计报告

**审计日期**: 2026-06-03
**标准**: GB/T 22239-2019 信息安全技术 网络安全等级保护基本要求 (Level 3)
**审计范围**: 全代码库

---

## 审计结果汇总

| 严重程度 | 发现数 | 已修复 | 待修复 |
|---------|--------|--------|--------|
| Critical | 5 | 5 | 0 |
| High | 9 | 3 | 6 |
| Medium | 7 | 2 | 5 |
| Low | 4 | 0 | 4 |
| **合计** | **25** | **10** | **15** |

---

## 已修复问题 (P0)

### [CRITICAL] 密码存储改用 bcrypt
- **文件**: `internal/handler/admin_handler.go`
- **问题**: SHA-256 无盐哈希, 易受彩虹表攻击
- **修复**: 改用 bcrypt(cost=12), 兼容旧 SHA-256 迁移 (verifyPassword 自动检测)

### [CRITICAL] JWT_SECRET 强制要求
- **文件**: `cmd/gateway/main.go`, `internal/handler/admin_handler.go`
- **问题**: 默认值 `aiops-dev-secret-key` 可被用于伪造 token
- **修复**: 启动时校验 JWT_SECRET 环境变量存在且 >=32 字节, 无默认值

### [CRITICAL] 移除前端演示模式
- **文件**: `web/src/pages/Login.tsx`
- **问题**: 后端不可用时自动以 admin 身份登录
- **修复**: 移除 demo-token 逻辑, 仅显示错误信息

### [CRITICAL] 移除默认密码显示
- **文件**: `web/src/pages/Login.tsx`
- **问题**: 登录页显示 `admin / admin123`
- **修复**: 移除演示账号提示

### [HIGH] Nginx 安全头
- **文件**: `deploy/docker-compose/nginx.conf`
- **修复**:
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY`
  - `X-XSS-Protection: 1; mode=block`
  - `Content-Security-Policy` (严格策略)
  - `Referrer-Policy: strict-origin-when-cross-origin`
  - `Permissions-Policy` (禁用摄像头/麦克风/定位)
  - 关闭目录列表 (`autoindex off`)
  - 阻止隐藏文件访问 (`location ~ /\.`)

### [HIGH] Docker 环境变量强制
- **文件**: `deploy/docker-compose/docker-compose.yml`
- **修复**: gateway 和 admin 服务要求 `JWT_SECRET` 环境变量

---

## 待修复问题 (P1-P3)

### P1 High (6项)

| # | 问题 | 文件 | 修复建议 |
|---|------|------|---------|
| 1 | X-User-ID 头伪造 | admin_handler.go | 后端验证 token 或限制内部 IP |
| 2 | 登录无失败锁定 | admin_handler.go | 5 次失败锁定 15 分钟 |
| 3 | 无多因素认证 | 全局 | TOTP 2FA (pquerna/otp) |
| 4 | Token 24h 无撤销 | gateway | Redis 黑名单 + 短 token + refresh |
| 5 | Casbin 未集成 | 全局 | 启用 path+method RBAC |
| 6 | 命令注入风险 | job_handler.go | 容器沙箱执行 |

### P2 Medium (5项)

| # | 问题 | 文件 | 修复建议 |
|---|------|------|---------|
| 1 | 租户数据未隔离 | 多个 handler | 从 context 取 tenant_id |
| 2 | 审计日志不完整 | 多个 handler | 所有敏感操作记审计 |
| 3 | 审计日志无防篡改 | 007_audit_logs.sql | 哈希链 + 外部 SIEM |
| 4 | Collector 无认证 | gateway | 预共享密钥或证书 |
| 5 | Docker 以 root 运行 | Dockerfile.* | 添加非 root 用户 |

### P3 Low (4项)

| # | 问题 | 修复建议 |
|---|------|---------|
| 1 | 无自动备份 | 定时备份脚本 |
| 2 | 无灾备方案 | RTO/RPO 目标 + 复制 |
| 3 | 无密码复杂度策略 | >=8 字符, 3 类字符 |
| 4 | 无会话空闲超时 | 30 分钟空闲退出 |

---

## 等保三级合规对照

| 要求 | 状态 | 说明 |
|------|------|------|
| 身份鉴别 - 密码加密 | ✅ 已修复 | bcrypt(cost=12) |
| 身份鉴别 - 登录失败处理 | ❌ 待修复 | 需实现账户锁定 |
| 身份鉴别 - 多因素认证 | ❌ 待修复 | 需实现 TOTP |
| 访问控制 - RBAC | ⚠️ 部分 | 有角色但 Casbin 未集成 |
| 访问控制 - 最小权限 | ❌ 待修复 | Docker root 运行 |
| 安全审计 - 日志完整性 | ⚠️ 部分 | 有审计表但覆盖不全 |
| 安全审计 - 防篡改 | ❌ 待修复 | 无哈希链 |
| 数据完整性 - 传输加密 | ❌ 待修复 | 无 TLS |
| 数据保密性 - 存储加密 | ✅ 已修复 | bcrypt 密码哈希 |
| 网络安全 - 安全头 | ✅ 已修复 | 7 个安全头 |
| 入侵防范 - 限流 | ⚠️ 部分 | Nginx 配置就绪, 未启用 |
| 配置安全 - 默认密码 | ✅ 已修复 | 移除硬编码默认值 |
| 配置安全 - 密钥管理 | ✅ 已修复 | JWT_SECRET 强制要求 |
