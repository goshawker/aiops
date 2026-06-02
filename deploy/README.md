# AIOps 平台部署方案

## 部署架构概览

```
┌──────────────────────────────────────────────────────────────────┐
│                        用户访问层                                 │
│   浏览器 (http://localhost:3000)     CLI/API (http://localhost:8080)│
└──────────────────────────────────────────────────────────────────┘
                               │
┌──────────────────────────────┴──────────────────────────────────┐
│                   网关层 (gateway :8080)                          │
│                   路由 / 鉴权 / 限流                              │
└────┬─────────┬──────────┬──────────┬──────────┬────────────────┘
     │         │          │          │          │
┌────┴──┐ ┌───┴───┐ ┌───┴───┐ ┌───┴───┐ ┌───┴────────┐
│query  │ │alert  │ │admin  │ │collect│ │job         │
│:8081  │ │:8082  │ │:8083  │ │:8084  │ │:8085       │
└──┬────┘ └───┬───┘ └───┬───┘ └───┬───┘ └─────┬──────┘
   │          │         │         │            │
┌──┴──────────┴─────────┴─────────┴────────────┴─────────────────┐
│                      AI 服务层 (Python)                          │
│  anomaly:5001  alert-agg:5003   llm:5004   rca:5005             │
└────────────────────────────────────────────────────────────────┘
```

---

## 一、部署模式选择

### 模式 A：单机一键部署（Docker Compose）

**适用场景**：开发测试、小团队试用（≤50 节点，≤100GB/天）

**硬件要求**：8C16G + 200GB SSD

**启动方式**：
```bash
# 1. 构建前端
make web-build

# 2. 构建 Docker 镜像
make docker-build

# 3. 启动所有服务
make docker-up

# 4. 查看日志
make docker-logs
```

**访问地址**：
- Web 控制台：http://localhost:3000
- API 网关：http://localhost:8080
- VictoriaMetrics：http://localhost:8428
- ClickHouse HTTP：http://localhost:8123

**包含组件**：
| 组件 | 说明 | 端口 |
|------|------|------|
| gateway | API 网关 | 8080 |
| query-service | 指标/日志查询 | 8081 |
| alert-engine | 告警引擎 | 8082 |
| admin-service | 管理后台 | 8083 |
| collector-service | 采集器管理 | 8084 |
| job-engine | 作业引擎 | 8085 |
| web | Nginx 前端 | 3000 |
| victoria-metrics | 时序数据库 | 8428 |
| clickhouse | 日志数据库 | 8123 |
| kafka + zookeeper | 消息队列 | 9092 |
| node-exporter | 主机指标采集 | 9100 |
| vmagent | 指标抓取代理 | 8429 |
| anomaly-service | 异常检测 | 5001 |
| alert-agg-service | 告警聚合 | 5003 |
| llm-service | LLM 服务 | 5004 |

> **最小化启动**：可用 `docker compose scale` 关闭不需要的服务（如 NebulaGraph、Kafka）

---

### 模式 B：Kubernetes 集群部署（Helm）

**适用场景**：生产环境（500+ 节点，≥1TB/天，需 99.9% 可用）

**硬件要求**：K8s 集群（3 master + 若干 worker），每 worker 16C64G + 2TB SSD

**部署方式**：
```bash
# 安装
helm install aiops ./deploy/helm/aiops --namespace aiops --create-namespace

# 更新
helm upgrade aiops ./deploy/helm/aiops --namespace aiops

# 卸载
helm uninstall aiops --namespace aiops
```

**Helm values 关键配置**（参考 `deploy/helm/aiops/values.yaml`）：
- `gateway.replicaCount`：网关副本数，建议 ≥2
- `victoriaMetrics.storage`：时序数据持久化大小
- `clickhouse.storage`：日志数据持久化大小
- `nebula.enabled`：是否启用图数据库

**高可用配置**：
- VictoriaMetrics 集群模式（vminsert / vmstorage / vmselect）
- ClickHouse 分片 + 副本
- Kafka 多 broker（≥3）
- API Gateway 多副本 + HPA

---

### 模式 C：边缘-中心混合部署

**适用场景**：多分支机构 / 边缘计算节点，数据需回传中心

**架构**：
```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  边缘节点 A   │    │  边缘节点 B   │    │  边缘节点 C   │
│ Agent+本地缓存│    │ Agent+本地缓存│    │ Agent+本地缓存│
└──────┬──────┘    └──────┬──────┘    └──────┬──────┘
       │                  │                  │
       └──────────────────┼──────────────────┘
                         │
                  ┌──────┴──────┐
                  │   中心集群   │
                  │ (完整部署)   │
                  └─────────────┘
```
- 边缘节点：部署轻量 Agent + Prometheus + VictoriaMetrics 单机
- 中心节点：全量部署，提供 AIOps 分析
- 网络策略：边缘主动 Push 数据到中心，支持断网续传

---

## 二、Agent 采集器部署方案

### 2.1 Agent 架构

```
┌─────────────────────────────────────┐
│         被监控主机                    │
│                                      │
│  ┌─────────────┐   ┌──────────────┐ │
│  │ AIOps Agent  │   │ Node Exporter│ │
│  │ (Go 二进制)   │   │ (Prometheus) │ │
│  │              │   │              │ │
│  │ 1.注册到中心  │   │ 9100: 指标  │ │
│  │ 2.定期心跳    │   │              │ │
│  │ 3.执行作业    │   │              │ │
│  │ 4.远程升级    │   │              │ │
│  └──────┬───────┘   └──────┬───────┘ │
│         │                  │          │
└─────────┼──────────────────┼──────────┘
          │                  │
          │           ┌──────┘
          ▼           ▼
   ┌─────────────────────────────┐
   │    Collector Service (:8084) │
   │    管理 Agent 注册/心跳/配置  │
   └─────────────────────────────┘
```

### 2.2 快速部署 Agent

#### 方式一：一键安装脚本

#### 下载方式

Agent 安装脚本和二进制文件通过以下途径提供：

| 来源 | URL | 说明 |
|------|-----|------|
| Nginx (部署服务器) | `http://<部署服务器IP>:3000/install.sh` | Docker Compose 部署时默认可用 |
| Collector 服务 | `http://<collector>:8084/api/v1/collectors/install.sh` | 通过 API 获取 |
| 本地文件 | `deploy/agent/install.sh` | 离线环境直接拷贝 |

Agent 二进制下载：

```bash
# 通过 Nginx
curl -sSL http://<部署服务器IP>:3000/agent/aiops-agent-linux-amd64 -o aiops-agent

# 通过 Collector API
curl -sSL http://<collector>:8084/api/v1/collectors/download/linux-amd64 -o aiops-agent
```

#### 在线安装（从部署服务器）

```bash
# <部署服务器IP> = Docker Compose 宿主机的 IP（非 localhost）
# 默认端口 3000（Nginx 前端）
curl -sSL http://<部署服务器IP>:3000/install.sh | bash -s -- \
  --collector http://<部署服务器IP>:8084 \
  --name $(hostname) \
  --tags '{"env":"prod","region":"cn-beijing"}'
```

> ⚠️ **不要在 Agent 目标机上使用 localhost**。`<部署服务器IP>` 是运行 AIOps 服务的机器 IP，Agent 需要通过网络连接到 Collector 服务（8084端口）。

#### 离线安装（无网络环境）

```bash
# 1. 在部署服务器上编译 Agent 二进制
make agent-build-linux-amd64

# 2. 将安装脚本和二进制拷贝到目标主机
scp deploy/agent/install.sh <目标主机>:/tmp/
scp bin/agent-linux-amd64 <目标主机>:/tmp/aiops-agent

# 3. SSH 到目标主机执行安装
ssh <目标主机> "bash /tmp/install.sh \
  --local-bin /tmp/aiops-agent \
  --collector http://<部署服务器IP>:8084 \
  --name $(hostname)"
```

脚本会自动完成：
1. 检测操作系统架构（x86_64 / ARM64）
2. 下载对应版本的 Agent 二进制
3. 安装为 systemd 服务
4. 启动并设置开机自启
5. 注册到 Collector 服务

#### 方式二：手动部署

```bash
# 1. 下载 Agent 二进制（在部署服务器上）
scp bin/agent <target-host>:/usr/local/bin/aiops-agent

# 2. 在目标主机上创建配置文件 /etc/aiops/agent.yaml
cat > /etc/aiops/agent.yaml << 'EOF'
collector_url: "http://<collector-host>:8084"
agent_name: "<hostname>"
interval: 30s
tags: '{"env":"prod"}'
EOF

# 3. 创建 systemd 服务
cat > /etc/systemd/system/aiops-agent.service << 'EOF'
[Unit]
Description=AIOps Agent
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/aiops-agent \
  --config /etc/aiops/agent.yaml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# 4. 启动
systemctl daemon-reload
systemctl enable --now aiops-agent
systemctl status aiops-agent
```

#### 方式三：Docker 化部署（适合容器化环境）

```yaml
# docker-compose.agent.yml
version: "3.8"
services:
  aiops-agent:
    image: aiops/agent:latest
    container_name: aiops-agent
    network_mode: host  # 采集主机指标需要
    environment:
      COLLECTOR_URL: "http://<collector-host>:8084"
      AGENT_NAME: "${HOSTNAME}"
      INTERVAL: "30s"
    volumes:
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /:/rootfs:ro
    restart: unless-stopped

  node-exporter:
    image: prom/node-exporter:v1.8.2
    container_name: node-exporter
    network_mode: host
    volumes:
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /:/rootfs:ro
    command:
      - "--path.procfs=/host/proc"
      - "--path.sysfs=/host/sys"
      - "--path.rootfs=/rootfs"
    restart: unless-stopped
```

### 2.3 Agent 生命周期管理

```
┌────────┐    ┌──────────┐    ┌───────────┐    ┌──────────┐
│ 注册    │───→│ 心跳上报  │───→│ 配置拉取  │───→│ 采集上报  │
│ POST   │    │ 每30s    │    │ 按需执行  │    │ /metrics │
│/collect│    │/heartbeat│    │/config    │    │          │
└────────┘    └──────────┘    └───────────┘    └──────────┘
```

1. **注册**：Agent 启动时向 Collector 服务注册，获取唯一 ID
2. **心跳**：每隔 `interval`（默认 30s）上报 CPU、内存等状态
3. **配置下发**：服务端可动态下发采集规则（支持 Prometheus scrape 配置）
4. **升级管理**：服务端推送新版本，Agent 自动下载并重启
5. **离线检测**：心跳超时 90s 标记为离线，服务端自动告警

### 2.4 Agent 二进制下载

Agent 二进制支持跨平台编译：

```bash
# 本地编译
make build-agent-linux-amd64
make build-agent-linux-arm64
make build-agent-darwin-arm64

# 交叉编译
GOOS=linux GOARCH=amd64 go build -o bin/agent-linux-amd64 ./cmd/agent
GOOS=linux GOARCH=arm64 go build -o bin/agent-linux-arm64 ./cmd/agent
```

| 平台 | 架构 | 二进制名 |
|------|------|----------|
| Linux | x86_64 | `agent-linux-amd64` |
| Linux | ARM64 (鲲鹏/飞腾) | `agent-linux-arm64` |
| Linux | LoongArch (龙芯) | `agent-linux-loong64` |
| Docker | 多架构 | `aiops/agent:latest` |

---

## 三、环境要求

### 3.1 操作系统

| 系统 | 版本 | 支持 |
|------|------|------|
| Ubuntu | 20.04+ | ✅ |
| CentOS / Rocky | 8+ | ✅ |
| 麒麟 V10 | SP1+ | ✅ 信创 |
| 统信 UOS | 1070+ | ✅ 信创 |
| 龙蜥 Anolis | 8+ | ✅ |

### 3.2 依赖组件

| 组件 | 版本要求 | 说明 |
|------|----------|------|
| Docker | 24.0+ | 容器运行时 |
| Docker Compose | 2.20+ | 单机编排 |
| Kubernetes | 1.28+ | 集群编排（可选） |
| Helm | 3.14+ | K8s 包管理（可选） |

### 3.3 硬件推荐

| 部署规模 | 节点数 | CPU | 内存 | 磁盘 |
|----------|--------|-----|------|------|
| 试用/小规模 | ≤50 | 8C | 16G | 200GB SSD |
| 中等规模 | 50-200 | 16C | 32G | 500GB SSD |
| 大规模 | 500+ | 32C×3 | 64G×3 | 2TB SSD×3 |

---

## 四、部署流程

### 4.1 Docker Compose 详细部署

```bash
# 1. 克隆项目
git clone <repo-url> && cd aiops

# 2. 构建 Go 服务
make build

# 3. 构建前端
make web-build

# 4. 构建 Docker 镜像
make docker-build

# 5. 配置环境变量（可选）
cp deploy/docker-compose/.env.example .env
# 编辑 .env 修改配置...

# 6. 启动
make docker-up

# 7. 验证
curl http://localhost:8080/api/v1/health
```

### 4.2 Helm 详细部署

```bash
# 1. 安装 Helm Chart
helm install aiops ./deploy/helm/aiops \
  --namespace aiops --create-namespace \
  --set gateway.replicaCount=2 \
  --set victoriaMetrics.storage=50Gi \
  --set clickhouse.storage=100Gi

# 2. 查看部署状态
kubectl get pods -n aiops -w
kubectl get svc -n aiops

# 3. 配置 Ingress
kubectl apply -f - << 'EOF'
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: aiops-ingress
  namespace: aiops
spec:
  rules:
  - host: aiops.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: aiops-gateway
            port:
              number: 8080
EOF
```

### 4.3 信创环境部署

```yaml
# 适配信创的 docker-compose 覆盖配置
# docker-compose.xinchuang.yml
services:
  victoria-metrics:
    image: victoriametrics/victoria-metrics:latest  # 支持 ARM64
  clickhouse:
    image: clickhouse/clickhouse-server:latest       # 支持 ARM64
  # 数据库替换：PostgreSQL → 达梦 DM8 / 人大金仓 KingbaseES
  # 对象存储替换：MinIO → 华为 OBS
```

---

## 五、运维指南

### 5.1 服务健康检查

```bash
# 检查各服务状态
curl http://localhost:8080/api/v1/health      # API 网关
curl http://localhost:8428/health              # VictoriaMetrics
curl http://localhost:8123/ping                # ClickHouse
curl http://localhost:8084/api/v1/status       # 采集器状态
```

### 5.2 日志查看

```bash
# Docker Compose
make docker-logs
docker compose -f deploy/docker-compose/docker-compose.yml logs -f gateway
docker compose -f deploy/docker-compose/docker-compose.yml logs -f anomaly-service

# Kubernetes
kubectl logs -n aiops -l app=aiops-gateway -f
```

### 5.3 数据备份

```bash
# VictoriaMetrics 快照
curl -X POST http://localhost:8428/snapshot/create

# ClickHouse 导出
clickhouse-client --query "SELECT * FROM aiops.logs" > logs_backup.tsv
```

### 5.4 升级流程

```bash
# Docker Compose 升级
git pull
make build
make web-build
make docker-build
make docker-down
make docker-up

# Helm 升级
helm upgrade aiops ./deploy/helm/aiops --namespace aiops
```

---

## 六、Agent 部署示例

### 示例：部署 Agent 到 5 台 Linux 服务器

```bash
# 在部署服务器上（192.168.1.100）：
TARGETS="192.168.1.10 192.168.1.11 192.168.1.12 192.168.1.20 192.168.1.21"

for HOST in $TARGETS; do
  ssh $HOST "curl -sSL http://192.168.1.100:8080/install.sh | bash -s -- \
    --collector http://192.168.1.100:8084 \
    --name \$(hostname) \
    --tags '{\"env\":\"prod\"}'"
done

# 验证所有 Agent 已连接
curl http://192.168.1.100:8084/api/v1/status
# 返回: {"total": 5, "online": 5, "offline": 0}
```

### 示例：Agent + Node Exporter 组合部署

```bash
# 在目标主机上同时部署 AIOps Agent 和 Node Exporter
docker compose -f /opt/aiops-agent/docker-compose.agent.yml up -d

# 查看采集状态
curl http://localhost:9100/metrics | head -20
curl http://localhost:<agent-port>/health
```
