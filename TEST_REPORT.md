# VigilOps 天枢 测试报告

**生成时间**: 2026-06-04
**测试范围**: 前端 (React/TS) + Go 后端 (Handler/Repo/Service) + Python AI 服务

---

## 汇总

| 层 | 测试文件数 | 用例数 | 通过 | 失败 | 通过率 | 耗时 |
|----|-----------|--------|------|------|--------|------|
| **Frontend (vitest)** | 8 | 47 | 47 | 0 | 100% | 0.7s |
| **Go Handler** | 5 | 34 | 34 | 0 | 100% | 3.1s |
| **Go Repo** | 6 | 44 | 44 | 0 | 100% | 0.5s |
| **Go Service** | 1 | 12 | 12 | 0 | 100% | 0.4s |
| **Python (pytest)** | 3 | 38 | 38 | 0 | 100% | 0.4s |
| **合计** | **23** | **175** | **175** | **0** | **100%** | **5.1s** |

---

## Frontend 测试详情 (47 tests)

### API 模块测试 (41 tests)

| 文件 | 用例数 | 状态 |
|------|--------|------|
| `api/__tests__/alerts.test.ts` | 11 | ✅ 全部通过 |
| `api/__tests__/admin.test.ts` | 12 | ✅ 全部通过 |
| `api/__tests__/metrics.test.ts` | 3 | ✅ 全部通过 |
| `api/__tests__/jobs.test.ts` | 6 | ✅ 全部通过 |
| `api/__tests__/anomaly.test.ts` | 3 | ✅ 全部通过 |
| `api/__tests__/traces.test.ts` | 4 | ✅ 全部通过 |
| `api/__tests__/rca.test.ts` | 2 | ✅ 全部通过 |

### Store 测试 (6 tests)

| 文件 | 用例数 | 状态 |
|------|--------|------|
| `store/__tests__/app.test.ts` | 6 | ✅ 全部通过 |

---

## Go Handler 测试详情 (34 tests)

| 文件 | 用例数 | 状态 |
|------|--------|------|
| `handler/admin_handler_test.go` | 16 | ✅ 全部通过 |
| `handler/alert_handler_test.go` | 10 | ✅ 全部通过 |
| `handler/job_handler_test.go` | 8 | ✅ 全部通过 |
| `handler/collector_handler_test.go` | 8 | ✅ 全部通过 |

### 测试覆盖内容
- Admin: 登录成功/失败/锁定/禁用、用户 CRUD、角色验证、自我提权防护、租户管理、Token 生成验证
- Alert: 规则 CRUD、事件/事件列表、确认/解决事件
- Job: 作业 CRUD、执行、危险命令验证、执行历史
- Collector: 注册/列表/删除/心跳/状态/下载验证

---

## Go Repo 测试详情 (44 tests)

| 文件 | 用例数 | 状态 |
|------|--------|------|
| `repo/admin_repo_test.go` | 12 | ✅ 全部通过 |
| `repo/alert_rule_repo_test.go` | 10 | ✅ 全部通过 |
| `repo/tenant_repo_test.go` | 7 | ✅ 全部通过 |
| `repo/collector_repo_test.go` | 8 | ✅ 全部通过 |
| `repo/job_repo_test.go` | 7 | ✅ 全部通过 |

### 测试覆盖内容
- SQLite 内存数据库 + 完整 schema 初始化
- CRUD 操作、唯一约束、分页、状态过滤
- 哈希链完整性 (审计日志)
- 命令验证 (validator 包)

---

## Go Service 测试详情 (12 tests)

| 文件 | 用例数 | 状态 |
|------|--------|------|
| `service/cron_scheduler_test.go` | 12 | ✅ 全部通过 |

### 测试覆盖内容
- Cron 表达式解析: `@hourly`, `@daily`, `@weekly` 别名
- 标准 5 字段 cron: `*/5 * * * *`, `0 9 * * 1-5`
- `matchCronField`: 通配符、步进、范围、列表、精确值、无效输入

---

## Python 测试详情 (38 tests)

| 文件 | 用例数 | 状态 |
|------|--------|------|
| `anomaly/test_detector.py` | 13 | ✅ 全部通过 |
| `llm/test_summary_engine.py` | 13 | ✅ 全部通过 |
| `rca/test_engine.py` | 12 | ✅ 全部通过 |

### 测试覆盖内容
- Anomaly: 3-sigma 检测、静态阈值、严重程度分级、标签一致性
- LLM: 健康摘要生成、状态升级逻辑、空数据处理
- RCA: 因果图发现、根因分析、相关性计算

---

## 测试基础设施

### Frontend
- **框架**: vitest + @testing-library/react + jsdom
- **配置**: `web/vitest.config.ts`
- **运行**: `cd web && npm test`

### Go
- **框架**: 标准库 testing + testify + SQLite 内存数据库
- **运行**: `JWT_SECRET=test-secret go test ./internal/... -v`

### Python
- **框架**: pytest
- **配置**: `ai/pytest.ini`
- **运行**: `cd ai && python3 -m pytest . -v`
