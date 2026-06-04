# VigilOps 天枢 平台安装部署指南

**版本**: 1.0.0
**更新日期**: 2026-06-03

---

## 目录

1. [架构总览](#1-架构总览)
2. [系统要求](#2-系统要求)
3. [快速部署（Docker Compose）](#3-快速部署docker-compose)
4. [详细配置说明](#4-详细配置说明)
5. [Agent 安装部署](#5-agent-安装部署)
6. [Kubernetes/Helm 部署](#6-kuberneteshelm-部署)
7. [离线部署](#7-离线部署)
8. [数据库初始化](#8-数据库初始化)
9. [运维管理](#9-运维管理)
10. [故障排查](#10-故障排查)

---

## 1. 架构总览

```
┌─────────────────────────────────────────────────────────────┐
│                      用户访问层                              │
│   浏览器 (http://<host>:3000)    API (http://<host>:8080)   │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────┼────────────────────────────────────┐
│              Nginx 反向代理 (:3000)                          │
│   / → 前端静态文件   /api/ → Gateway   /agent/ → 安装脚本    │
└────────┬───────────────┼────────────────────────────────────┘
         │               │
┌────────┘    ┌──────────┴──────────────────────────────────┐
│             │          API Gateway (:8080)                 │
│             │      路由 / 认证 / 限流 / 日志               │
│             └──┬────┬────┬────┬────┬────┬─────────────────┘
│                │    │    │    │    │    │
│  ┌─────────────┤    │    │    │    │    ├──────────────────┐
│  │query   :8081│    │    │    │    │    │admin       :8083 │
│  │              │    │    │    │    │    │collector   :8084 │
│  │alert    :8082│    │    │    │    │    │job         :8085 │
│  └──────────────┘    │    │    │    │    └──────────────────┘
│                      │    │    │    │
│  ┌───────────────────┤    │    │    ├──────────────────────┐
│  │anomaly      :5001 │    │    │    │llm             :5004 │
│  │alert-agg    :5003 │    │    │    │rca             :5005 │
│  └───────────────────┘    │    │    └──────────────────────┘
│                           │    │
│  ┌────────────────────────┤    ├────────────────────────────┐
│  │   VictoriaMetrics :8428│    │ClickHouse :8123/:9000      │
│  │   (时序指标存储)        │    │(日志/链路存储)              │
│  └────────────────────────┘    └────────────────────────────┘
│                               │
│  ┌────────────────────────────┤
│  │   Kafka (:9092) + Zookeeper (:2181)                      │
│  │   (消息总线: 指标/日志/告警/事件)                          │
│  └──────────────────────────────────────────────────────────┘
│
│  ┌──────────────────────────────────────────────────────────┐
│  │   远程主机 Agent × N                                     │
│  │   指标采集 → Push 到 Collector → VMAgent → VictoriaMetrics│
│  └──────────────────────────────────────────────────────────┘
└──────────────────────────────────────────────────────────────┘
```

### 服务清单

| 服务 | 端口 | 语言 | 用途 |
|------|------|------|------|
| **gateway** | 8080 | Go | API 网关，路由/认证/限流 |
| **query** | 8081 | Go | 指标/日志查询 |
| **alert** | 8082 | Go | 告警引擎 + 规则评估 |
| **admin** | 8083 | Go | 用户/租户/审计管理 |
| **collector** | 8084 | Go | Agent 注册/心跳/配置分发 |
| **job** | 8085 | Go | 定时任务/远程执行 |
| **anomaly** | 5001 | Python | AI 异常检测 |
| **alert-agg** | 5003 | Python | 告警聚合/关联 |
| **llm** | 5004 | Python | AI 健康摘要 |
| **rca** | 5005 | Python | 根因分析 |
| **web** | 3000 | Nginx | 前端 + 反向代理 |
| **VictoriaMetrics** | 8428 | - | 时序指标存储 |
| **ClickHouse** | 8123/9000 | - | 日志/链路存储 |
| **Kafka** | 9092 | - | 消息总线 |
| **Node Exporter** | 9100 | - | 主机指标采集 |
| **VMAgent** | 8429 | - | Prometheus 抓取代理 |

---

## 2. 系统要求

### 2.1 硬件要求

| 部署模式 | CPU | 内存 | 磁盘 | 适用场景 |
|---------|-----|------|------|---------|
| **最小部署** | 4 核 | 8 GB | 100 GB SSD | 开发/测试 |
| **标准部署** | 8 核 | 16 GB | 200 GB SSD | 小型团队 (≤50 节点) |
| **生产部署** | 16 核 | 64 GB | 2 TB SSD | 大规模 (500+ 节点) |

### 2.2 软件要求

| 软件 | 最低版本 | 用途 |
|------|---------|------|
| Docker | 20.10+ | 容器运行时 |
| Docker Compose | 2.0+ | 编排工具 |
| Git | 2.30+ | 源码获取 |
| Node.js | 18+ | 前端构建 |
| Go | 1.22+ | 后端构建 |
| Python | 3.10+ | AI 服务 |

### 2.3 网络要求

| 端口 | 方向 | 说明 |
|------|------|------|
| 3000 | 入站 | 用户访问前端 + Agent 下载 |
| 8080 | 入站 | API 访问 |
| 8084 | 入站 | Agent 注册/心跳 |

---

## 3. 快速部署（Docker Compose）

### 3.1 获取源码

```bash
git clone https://github.com/goshawker/aiops.git
cd aiops
```

### 3.2 一键部署

```bash
# 构建前端
make web-build

# 构建所有 Docker 镜像
make docker-build

# 启动全部服务
make docker-up
```

### 3.3 验证部署

```bash
# 检查所有容器状态
make docker-ps

# 预期输出：所有服务 Up 状态
# NAME               STATUS          PORTS
# aiops-gateway      Up              0.0.0.0:8080->8080/tcp
# aiops-query        Up              0.0.0.0:8081->8081/tcp
# aiops-alert        Up              0.0.0.0:8082->8082/tcp
# aiops-admin        Up              0.0.0.0:8083->8083/tcp
# aiops-collector    Up              0.0.0.0:8084->8084/tcp
# aiops-job          Up              0.0.0.0:8085->8085/tcp
# aiops-anomaly      Up              0.0.0.0:5001->5001/tcp
# aiops-llm          Up              0.0.0.0:5004->5004/tcp
# aiops-web          Up              0.0.0.0:3000->80/tcp
# victoria-metrics   Up              0.0.0.0:8428->8428/tcp
# clickhouse         Up              0.0.0.0:8123->8123/tcp
# kafka              Up              0.0.0.0:9092->9092/tcp
# node-exporter      Up              0.0.0.0:9100->9100/tcp
```

### 3.4 访问系统

- **前端**: http://localhost:3000
- **默认账号**: admin / admin123
- **API 网关**: http://localhost:8080

### 3.5 健康检查

```bash
# 检查 Gateway
curl http://localhost:8080/api/v1/health

# 检查各服务
curl http://localhost:8081/api/v1/health  # query
curl http://localhost:8082/api/v1/health  # alert
curl http://localhost:8083/api/v1/health  # admin
curl http://localhost:8084/api/v1/health  # collector
curl http://localhost:8085/api/v1/health  # job

# 检查 AI 服务
curl http://localhost:5001/health  # anomaly
curl http://localhost:5004/health  # llm
```

---

## 4. 详细配置说明

### 4.1 服务配置文件

所有 Go 服务配置位于 `configs/` 目录：

| 文件 | 服务 | 关键配置 |
|------|------|---------|
| `gateway.yaml` | Gateway | 上游路由、JWT 密钥、限流 (100 rps) |
| `query.yaml` | Query | VictoriaMetrics/ClickHouse 连接 |
| `alert.yaml` | Alert | Kafka topic、通知渠道 (钉钉/邮件/Webhook) |
| `admin.yaml` | Admin | 数据库路径、JWT 密钥 |
| `collector.yaml` | Collector | 心跳间隔 (30s)、离线超时 (90s) |
| `job.yaml` | Job | 并发数 (10)、步骤超时 (300s) |

### 4.2 环境变量覆盖

Docker Compose 支持通过环境变量覆盖配置：

```bash
# .env 文件
GATEWAY_JWT_SECRET=your-secret-key-here
CLICKHOUSE_HOST=clickhouse
KAFKA_BROKERS=kafka:9092
VICTORIA_METRICS_URL=http://victoria-metrics:8428
```

### 4.3 通知渠道配置

在 `configs/alert.yaml` 中配置：

```yaml
notification:
  dingtalk:
    enabled: true
    webhook: "https://oapi.dingtalk.com/robot/send?access_token=YOUR_TOKEN"
  email:
    enabled: true
    smtp_host: "smtp.example.com"
    smtp_port: 465
    from: "aiops@example.com"
  webhook:
    enabled: false
    url: ""
```

### 4.4 数据保留策略

| 数据类型 | 默认保留 | 存储位置 |
|---------|---------|---------|
| 指标数据 | 30 天 | VictoriaMetrics (`-retentionPeriod=30d`) |
| 日志数据 | 30 天 | ClickHouse (TTL 30 天) |
| 链路数据 | 7 天 | ClickHouse (TTL 7 天) |
| 告警事件 | 永久 | SQLite |
| 审计日志 | 永久 | SQLite |

---

## 5. Agent 安装部署

Agent 部署在远程主机上，负责采集指标并上报到 VigilOps 平台。

### 5.1 在线安装（推荐）

**一键安装命令**（在目标主机上执行）：

```bash
curl -sSL http://<VigilOps服务器IP>:3000/install.sh | bash -s -- \
  --collector http://<VigilOps服务器IP>:8084 \
  --name "web-server-01" \
  --tags '{"env":"production","region":"cn-east"}'
```

**参数说明**：

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--collector` | `http://localhost:8084` | Collector 服务地址 |
| `--name` | `$(hostname)` | Agent 名称（默认使用主机名） |
| `--tags` | `{}` | JSON 标签（用于分组/过滤） |
| `--version` | `latest` | Agent 版本 |
| `--install-dir` | `/usr/local/bin` | 安装目录 |

**安装过程**：
1. 自动检测操作系统 (Linux/Darwin) 和架构 (amd64/arm64/loong64)
2. 从 Nginx 下载对应平台的 Agent 二进制文件
3. 写入配置 `/etc/aiops/agent.yaml`
4. 创建 systemd 服务 `aiops-agent`
5. 启动服务并设置开机自启

### 5.2 离线安装

适用于无法访问外网的目标主机。

**步骤 1：在有网环境构建 Agent**

```bash
# 构建所有平台的 Agent
make agent-build-all

# 产物在 bin/ 目录：
# bin/agent-linux-amd64
# bin/agent-linux-arm64
# bin/agent-linux-loong64
# bin/agent-darwin-amd64
# bin/agent-darwin-arm64
```

**步骤 2：传输到目标主机**

```bash
# 传输安装脚本和二进制
scp deploy/agent/install.sh bin/agent-linux-amd64 user@target-host:/tmp/
```

**步骤 3：在目标主机执行离线安装**

```bash
bash /tmp/install.sh \
  --local-bin /tmp/agent-linux-amd64 \
  --collector http://<VigilOps服务器IP>:8084 \
  --name "offline-server-01"
```

### 5.3 批量部署

**方式 1：SSH 批量执行**

```bash
# 使用 Makefile
make deploy-agent HOST=user@host1 COLLECTOR=http://aiops-server:8084
make deploy-agent HOST=user@host2 COLLECTOR=http://aiops-server:8084
```

**方式 2：Ansible 批量部署**

```yaml
# playbook.yml
- hosts: target_hosts
  become: yes
  tasks:
    - name: Install VigilOps Agent
      shell: |
        curl -sSL http://{{ aiops_server }}:3000/install.sh | bash -s -- \
          --collector http://{{ aiops_server }}:8084 \
          --name "{{ inventory_hostname }}" \
          --tags '{"env":"{{ env }}"}'
```

### 5.4 Agent 管理

```bash
# 查看 Agent 状态
systemctl status aiops-agent

# 查看 Agent 日志
journalctl -u aiops-agent -f

# 重启 Agent
systemctl restart aiops-agent

# 停止 Agent
systemctl stop aiops-agent

# 禁用开机自启
systemctl disable aiops-agent
```

### 5.5 Agent 配置文件

路径：`/etc/aiops/agent.yaml`

```yaml
collector_url: "http://aiops-server:8084"
agent_name: "web-server-01"
interval: 30s
tags: '{"env":"production","region":"cn-east"}'
```

### 5.6 支持的平台

| 操作系统 | 架构 | 二进制文件名 |
|---------|------|-------------|
| Linux | x86_64 (amd64) | `agent-linux-amd64` |
| Linux | ARM64 (鲲鹏/飞腾) | `agent-linux-arm64` |
| Linux | LoongArch (龙芯) | `agent-linux-loong64` |
| macOS | x86_64 (Intel) | `agent-darwin-amd64` |
| macOS | ARM64 (Apple Silicon) | `agent-darwin-arm64` |

---

## 6. Kubernetes/Helm 部署

### 6.1 前置条件

- Kubernetes 1.24+
- Helm 3.12+
- kubectl 配置正确

### 6.2 安装

```bash
# 添加 Helm 仓库（如果已发布）
# helm repo add aiops https://charts.aiops.io

# 本地安装
cd deploy/helm/aiops

# 自定义 values
cat > my-values.yaml << EOF
gateway:
  replicaCount: 2
  jwtSecret: "your-production-secret"

victoriaMetrics:
  storage: 50Gi

clickhouse:
  storage: 100Gi

web:
  ingress:
    enabled: true
    host: aiops.example.com
EOF

# 安装
helm install aiops . -f my-values.yaml -n aiops --create-namespace
```

### 6.3 关键 Helm Values

```yaml
# deploy/helm/aiops/values.yaml
gateway:
  replicaCount: 2
  jwtSecret: "change-me-in-production"

victoriaMetrics:
  storage: 10Gi        # 生产环境建议 50Gi+
  retentionPeriod: 30d

clickhouse:
  storage: 20Gi        # 生产环境建议 100Gi+
  logRetentionDays: 30
  traceRetentionDays: 7

nebula:
  enabled: false       # 图数据库（可选，用于拓扑分析）

anomaly:
  enabled: true
  replicaCount: 1

llm:
  enabled: true
  replicaCount: 1
```

---

## 7. 离线部署

适用于无法访问外网的生产环境。

### 7.1 构建离线包

```bash
# 在有网环境执行
make offline-build

# 产物：deploy/offline/aiops-offline-<version>.tar.gz
# 包含：Docker 镜像 + Agent 二进制 + 配置文件 + 安装脚本
```

### 7.2 传输到目标环境

```bash
scp deploy/offline/aiops-offline-*.tar.gz user@offline-host:/opt/
```

### 7.3 在离线环境安装

```bash
cd /opt
tar xzf aiops-offline-*.tar.gz
cd aiops-offline

# 加载 Docker 镜像
bash load-images.sh

# 启动服务
docker compose up -d

# 安装 Agent（使用本地二进制）
bash agent/install.sh --local-bin agent/agent-linux-amd64 --collector http://localhost:8084
```

---

## 8. 数据库初始化

### 8.1 SQLite（自动初始化）

SQLite 数据库在 admin 服务首次启动时自动创建。初始化脚本位于 `deploy/sql/`：

| 脚本 | 表 | 说明 |
|------|-----|------|
| `001_tenants.sql` | tenants | 多租户 |
| `002_users.sql` | users, casbin_rule | 用户 + RBAC |
| `003_collectors.sql` | collectors, collector_configs | Agent 管理 |
| `004_alert_rules.sql` | alert_rules | 告警规则 |
| `005_alert_events.sql` | alert_events, incidents | 告警事件 |
| `006_jobs.sql` | jobs, job_executions | 定时任务 |
| `007_audit_logs.sql` | audit_logs | 审计日志 |
| `008_system_config.sql` | system_config | 系统配置 |

**默认管理员账号**: admin / admin123

### 8.2 ClickHouse

```bash
# 手动初始化
clickhouse-client --multiquery < deploy/clickhouse/001_logs.sql
```

**表结构**：
- `aiops.logs` — 日志表 (MergeTree, TTL 30 天)
- `aiops.traces` — 链路表 (MergeTree, TTL 7 天)

---

## 9. 运维管理

### 9.1 常用命令

```bash
# 查看服务状态
make docker-ps

# 查看日志
make docker-logs                    # 所有服务
docker compose logs -f gateway      # 单个服务

# 重启服务
make docker-restart                 # 全部重启
docker compose restart gateway      # 单个重启

# 更新配置后重启
docker compose restart alert        # 修改 alert.yaml 后
```

### 9.2 备份与恢复

```bash
# 备份 SQLite
cp aiops.db aiops.db.bak.$(date +%Y%m%d)

# 备份 VictoriaMetrics 数据
docker exec victoria-metrics tar czf /tmp/vm-backup.tar.gz /victoria-metrics-data
docker cp victoria-metrics:/tmp/vm-backup.tar.gz ./vm-backup.tar.gz

# 备份 ClickHouse
docker exec clickhouse clickhouse-client --query "BACKUP DATABASE aiops TO Disk('default', '/tmp/ch-backup')"
```

### 9.3 监控

VigilOps 自身指标暴露在各服务的 `/metrics` 端点：

```bash
# 查看 Gateway 指标
curl http://localhost:8080/metrics

# VMAgent 已配置自动抓取所有服务的 /metrics
# 在 VictoriaMetrics 中查询：
# {job="aiops-services"}
```

### 9.4 日志查看

```bash
# 实时日志
docker compose logs -f --tail=100 gateway

# 按服务过滤
docker compose logs alert --since 1h

# 在 VigilOps 前端查看
# 访问 http://localhost:3000 → 可观测性 → 日志
```

---

## 10. 故障排查

### 10.1 服务启动失败

```bash
# 查看容器日志
docker compose logs <service-name> --tail=50

# 常见原因：
# 1. 端口被占用 → netstat -tlnp | grep <port>
# 2. 依赖服务未就绪 → 检查 Kafka/VM/ClickHouse 状态
# 3. 配置文件错误 → 检查 configs/*.yaml 语法
```

### 10.2 Agent 无法注册

```bash
# 在目标主机检查 Agent 日志
journalctl -u aiops-agent -f

# 检查网络连通性
curl http://<VigilOps服务器>:8084/api/v1/health

# 检查防火墙
# 需要开放: 8084 (collector), 3000 (agent下载)
```

### 10.3 前端无法访问

```bash
# 检查 Nginx 容器
docker compose logs web --tail=20

# 检查 Gateway 是否可达
curl -I http://localhost:8080/api/v1/health

# 检查前端构建产物
ls -la web/dist/
```

### 10.4 指标数据缺失

```bash
# 检查 VMAgent 是否在抓取
curl http://localhost:8429/targets

# 检查 VictoriaMetrics 是否有数据
curl "http://localhost:8428/api/v1/query?query=up"

# 检查 Node Exporter
curl http://localhost:9100/metrics | head -20
```

### 10.5 告警不触发

```bash
# 检查告警规则是否启用
curl -H "X-User-ID: 1" -H "X-Tenant-ID: 1" http://localhost:8080/api/v1/alerts/rules

# 检查 alert 服务日志
docker compose logs alert --tail=50

# 检查 Kafka topic
docker exec kafka kafka-topics --bootstrap-server localhost:9092 --list
```

### 10.6 磁盘空间不足

```bash
# 查看 Docker 磁盘使用
docker system df

# 清理未使用资源
docker system prune -f

# 清理旧镜像
docker image prune -a -f --filter "until=720h"  # 30天前
```

---

## 附录 A：端口速查表

| 端口 | 服务 | 说明 |
|------|------|------|
| 3000 | web (nginx) | 前端 UI + Agent 下载 |
| 5001 | anomaly | 异常检测 API |
| 5003 | alert-agg | 告警聚合 API |
| 5004 | llm | LLM 服务 API |
| 5005 | rca | 根因分析 API |
| 8080 | gateway | API 网关 |
| 8081 | query | 查询服务 |
| 8082 | alert | 告警引擎 |
| 8083 | admin | 管理服务 |
| 8084 | collector | Agent 管理 |
| 8085 | job | 任务引擎 |
| 8123 | clickhouse | ClickHouse HTTP |
| 8428 | victoria-metrics | 时序存储 |
| 8429 | vmagent | 抓取代理 |
| 9000 | clickhouse | ClickHouse Native |
| 9092 | kafka | 消息总线 |
| 9100 | node-exporter | 主机指标 |

## 附录 B：Makefile 命令速查

| 命令 | 说明 |
|------|------|
| `make build` | 构建所有 Go 服务 |
| `make test` | 运行 Go 测试 |
| `make web-build` | 构建前端 |
| `make docker-build` | 构建 Docker 镜像 |
| `make docker-up` | 启动所有服务 |
| `make docker-down` | 停止所有服务 |
| `make docker-logs` | 查看日志 |
| `make agent-build-all` | 构建全平台 Agent |
| `make deploy-agent HOST=user@host` | 远程部署 Agent |
| `make offline-build` | 构建离线包 |
| `make clean` | 清理构建产物 |

## 附录 C：默认账号

| 账号 | 密码 | 角色 | 说明 |
|------|------|------|------|
| admin | admin123 | admin | 系统管理员（首次登录后请修改密码） |
