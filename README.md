# AIOps Platform

智能运维平台 — 集成指标监控、日志分析、链路追踪、异常检测、告警管理、根因分析于一体。

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go)](https://go.dev/)
[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react)](https://react.dev/)
[![Python](https://img.shields.io/badge/Python-3.12-3776AB?logo=python)](https://www.python.org/)
[![License](https://img.shields.io/badge/License-GPL--3.0%20with%20Commercial%20Exception-blue.svg)](LICENSE)

---

## 特性

- **全栈可观测性** — 指标 (VictoriaMetrics)、日志 (ClickHouse)、链路 (OpenTelemetry) 统一查询
- **AI 驱动运维** — 在线学习异常检测 (River)、因果图根因分析、LLM 健康摘要
- **告警引擎** — 规则评估 + 告警聚合 + 事件关联 + 多渠道通知 (钉钉/邮件/Webhook)
- **自动化作业** — Cron 调度 + Shell/HTTP 执行 + 远程 Agent 管理
- **多租户 RBAC** — 租户隔离 + 角色权限 (admin/operator/viewer)
- **信创适配** — 达梦/人大金仓数据库适配器 + LoongArch/ARM64 架构支持
- **一键部署** — Docker Compose / Kubernetes Helm / 离线安装包

## 架构

```
浏览器 (3000)  ──→  Nginx  ──→  Gateway (8080)  ──→  Go 微服务 ×6
                                    │
                                    ├── Query    (8081)  指标/日志查询
                                    ├── Alert    (8082)  告警引擎
                                    ├── Admin    (8083)  用户/租户管理
                                    ├── Collector(8084)  Agent 管理
                                    └── Job      (8085)  定时任务
                                    │
                                    ├── Anomaly  (5001)  AI 异常检测
                                    ├── AlertAgg (5003)  告警聚合
                                    ├── LLM      (5004)  健康摘要
                                    └── RCA      (5005)  根因分析

VictoriaMetrics (:8428)  ←  VMAgent ←  Agent ×N (远程主机)
ClickHouse     (:8123)  ←  Kafka
SQLite         (本地)   ←  Admin/Alert/Collector/Job
```

## 快速开始

### 前置条件

- Docker 20.10+ & Docker Compose 2.0+
- Git

### 一键部署

```bash
git clone https://github.com/goshawker/aiops.git
cd aiops

# 构建并启动
make web-build
make docker-build
make docker-up
```

### 访问

- **前端**: http://localhost:3000
- **API**: http://localhost:8080
- **默认账号**: `admin` / `admin123`

### 健康检查

```bash
make docker-ps          # 查看所有服务状态
curl localhost:8080/api/v1/health
```

## Agent 安装

在远程主机上采集系统指标：

```bash
# 在线安装
curl -sSL http://<server>:3000/install.sh | bash -s -- \
  --collector http://<server>:8084 \
  --name "web-server-01"

# 离线安装
bash install.sh --local-bin ./agent-linux-amd64 --collector http://<server>:8084
```

支持平台：Linux (amd64/arm64/loong64)、macOS (amd64/arm64)

## 技术栈

| 层 | 技术 |
|----|------|
| **前端** | React 18, TypeScript, Ant Design 5, ECharts, Zustand, Vite |
| **后端** | Go 1.22, Gin, SQLite, ClickHouse, Kafka |
| **AI 服务** | Python 3.12, River (在线学习), NumPy, PyTorch |
| **存储** | VictoriaMetrics (指标), ClickHouse (日志/链路), SQLite (元数据) |
| **部署** | Docker Compose, Kubernetes Helm, Nginx |

## 项目结构

```
aiops/
├── cmd/                    # Go 微服务入口
│   ├── gateway/            #   API 网关
│   ├── query/              #   查询服务
│   ├── alert/              #   告警引擎
│   ├── admin/              #   管理服务
│   ├── collector/          #   Agent 管理
│   ├── job/                #   任务引擎
│   └── agent/              #   数据采集 Agent
├── internal/               # Go 共享库
│   ├── handler/            #   HTTP 处理器
│   ├── repo/               #   数据访问层
│   ├── service/            #   业务逻辑层
│   └── model/              #   数据模型
├── web/                    # React 前端
│   └── src/
│       ├── api/            #   API 模块 (typed)
│       ├── pages/          #   页面组件 (11 个)
│       ├── components/     #   通用组件
│       └── store/          #   状态管理
├── ai/                     # Python AI 服务
│   ├── anomaly/            #   异常检测
│   ├── rca/                #   根因分析
│   └── llm/                #   LLM 服务
├── deploy/                 # 部署配置
│   ├── docker-compose/     #   Docker Compose
│   ├── helm/               #   Kubernetes Helm
│   ├── sql/                #   数据库 Schema
│   └── agent/              #   Agent 安装脚本
├── configs/                # 服务配置文件
└── api/proto/              # gRPC Proto 定义
```

## 开发

```bash
# 后端
make build                  # 构建所有 Go 服务
make test                   # 运行 Go 测试

# 前端
cd web && npm install       # 安装依赖
npm run dev                 # 开发服务器
npm test                    # 运行测试

# AI 服务
cd ai && pip install -r requirements.txt
python -m pytest . -v       # 运行测试
```

## 测试

| 层 | 框架 | 用例数 | 通过率 |
|----|------|--------|--------|
| 前端 | vitest + @testing-library/react | 47 | 100% |
| 后端 | Go testing + testify | 12 | 100% |
| AI | pytest | 38 | 100% |
| **合计** | | **97** | **100%** |

```bash
# 运行全部测试
cd web && npm test          # 前端
go test ./internal/... -v   # 后端
cd ai && python -m pytest . -v  # AI
```

## 文档

- [安装部署指南](deploy/DEPLOYMENT_GUIDE.md) — Docker Compose / K8s / Agent / 离线部署
- [测试报告](TEST_REPORT.md) — 测试覆盖详情

## 贡献

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'feat: add amazing feature'`)
4. 推送分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 许可证

本项目采用 **GPL-3.0 + 商业豁免条款** 双重许可。详见 [LICENSE](LICENSE)。

- 开源使用：遵循 GPL-3.0 条款
- 商业使用：年收入低于 10 万美元的组织可免费商用；超出范围需获取商业许可
