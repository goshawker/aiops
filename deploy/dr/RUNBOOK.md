# AIOps 灾备运维手册

## RTO/RPO 目标

| 指标 | 目标 | 说明 |
|------|------|------|
| **RTO** (恢复时间目标) | < 30 分钟 | 从故障到服务恢复 |
| **RPO** (数据恢复点) | < 24 小时 | 最多丢失 24 小时数据 |
| **可用性目标** | 99.5% | 单节点部署 |

---

## 架构模式

### 模式 A: 单节点 + 异地备份 (推荐中小规模)

```
[主节点]  ──rsync/ssh──>  [备份节点]
  AIOps 全栈               备份存储 + 冷备
```

- 每天自动备份 (backup.sh)
- 每天复制到备份节点 (replicate.sh)
- 故障时在备份节点恢复

### 模式 B: 双节点热备 (中大规模)

```
[主节点]                    [备节点]
  Gateway (active)           Gateway (standby)
  Admin/Alert/Job (active)   Admin/Alert/Job (standby)
  VictoriaMetrics (active)   VictoriaMetrics (replica)
  ClickHouse (active)        ClickHouse (replica)
```

- 共享存储或实时复制
- 负载均衡器自动切换
- 数据库主从复制

### 模式 C: Kubernetes 多副本 (大规模)

```
K8s Cluster
├── gateway (2 replicas)
├── admin (2 replicas)
├── alert (2 replicas)
├── victoria-metrics (StatefulSet, 3 replicas)
├── clickhouse (StatefulSet, 2 replicas)
└── web (2 replicas)
```

---

## 故障恢复流程

### 场景 1: 单个服务崩溃

```bash
# 自动恢复 (healthcheck.sh --fix 会自动执行)
bash deploy/dr/healthcheck.sh --fix

# 手动恢复
docker compose -f deploy/docker-compose/docker-compose.yml restart <service>
```

### 场景 2: 主机故障 (单节点)

```bash
# 1. 在备份节点安装 AIOps
git clone https://github.com/goshawker/aiops.git
cd aiops
make web-build && make docker-build

# 2. 从备份恢复数据
bash deploy/backup/restore.sh --sqlite /opt/aiops/backups/sqlite/latest.db
bash deploy/backup/restore.sh --clickhouse /opt/aiops/backups/clickhouse/latest.tar.gz
bash deploy/backup/restore.sh --vm /opt/aiops/backups/victoriametrics/latest.tar.gz

# 3. 启动服务
export JWT_SECRET="your-production-secret"
make docker-up

# 4. 验证
bash deploy/dr/healthcheck.sh
```

### 场景 3: 数据损坏

```bash
# 1. 停止受影响的服务
docker compose stop admin alert

# 2. 恢复备份
bash deploy/backup/restore.sh --sqlite /opt/aiops/backups/sqlite/aiops_20260603.db

# 3. 重启
docker compose start admin alert

# 4. 验证数据完整性
bash deploy/dr/healthcheck.sh
```

### 场景 4: 数据库主从切换 (模式 B)

```bash
# 1. 停止主节点写入
docker compose stop gateway

# 2. 提升备节点为主节点
ssh standby "docker compose -f /opt/aiops/deploy/docker-compose/docker-compose.yml up -d"

# 3. 更新 DNS/负载均衡器指向备节点

# 4. 验证
curl https://standby-ip/health
```

---

## 监控与告警

### 持续健康监控

```bash
# 每 30 秒检查一次，异常自动告警
bash deploy/dr/healthcheck.sh --watch 30 --fix
```

### Cron 定时任务

```bash
# 每 5 分钟健康检查
*/5 * * * * /opt/aiops/deploy/dr/healthcheck.sh --fix >> /var/log/aiops-healthcheck.log 2>&1

# 每天凌晨 2 点备份
0 2 * * * /opt/aiops/deploy/backup/backup.sh --retention 30 >> /var/log/aiops-backup.log 2>&1

# 每天凌晨 3 点复制到备份节点
0 3 * * * /opt/aiops/deploy/dr/replicate.sh --to standby@192.168.1.200 >> /var/log/aiops-replicate.log 2>&1
```

### 告警 Webhook

```bash
# 设置钉钉/企业微信 Webhook
export ALERT_WEBHOOK="https://oapi.dingtalk.com/robot/send?access_token=xxx"
bash deploy/dr/healthcheck.sh --watch 60
```

---

## 备份策略

| 数据类型 | 频率 | 保留期 | 存储位置 |
|---------|------|--------|---------|
| SQLite | 每天 | 30 天 | 本地 + 备份节点 |
| ClickHouse (日志) | 每天 | 30 天 | 本地 + 备份节点 |
| ClickHouse (链路) | 每天 | 7 天 | 本地 + 备份节点 |
| VictoriaMetrics | 每天 | 30 天 | 本地 + 备份节点 |

---

## 备份恢复验证

每月执行一次恢复演练：

```bash
# 1. 在测试环境恢复最新备份
bash deploy/backup/restore.sh --sqlite /opt/aiops/backups/sqlite/latest.db

# 2. 启动服务验证
docker compose up -d admin
curl http://localhost:8083/health

# 3. 验证数据完整性
# - 检查用户表: SELECT COUNT(*) FROM users
# - 检查审计日志: SELECT COUNT(*) FROM audit_logs
# - 检查哈希链: SELECT record_hash FROM audit_logs ORDER BY id DESC LIMIT 5

# 4. 记录恢复时间
echo "Recovery time: $(date)" >> /var/log/aiops-dr-drill.log
```

---

## 联系方式

| 角色 | 联系方式 |
|------|---------|
| 运维负责人 | [待填写] |
| 数据库管理员 | [待填写] |
| 安全负责人 | [待填写] |
