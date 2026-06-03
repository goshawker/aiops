# AIOps 测试报告

**生成时间**: 2026-06-03
**测试范围**: 前端 (React/TS) + Go 后端 + Python AI 服务

---

## 汇总

| 层 | 测试文件数 | 用例数 | 通过 | 失败 | 通过率 |
|----|-----------|--------|------|------|--------|
| **Frontend (vitest)** | 8 | 47 | 47 | 0 | 100% |
| **Go (go test)** | 1 | 12 | 12 | 0 | 100% |
| **Python (pytest)** | 3 | 38 | 38 | 0 | 100% |
| **合计** | **12** | **97** | **97** | **0** | **100%** |

> 注: `tests/` 目录下有 9 个旧测试存在预存失败（mock 接口不匹配），不在本次修复范围内。

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

### 测试覆盖内容
- API 模块: 验证每个 API 方法调用正确的 HTTP 方法、路径和参数
- Store: 初始状态、主题切换、侧边栏折叠、认证/token 持久化、登出清理

---

## Go 测试详情 (12 tests)

| 文件 | 用例数 | 状态 |
|------|--------|------|
| `internal/service/cron_scheduler_test.go` | 12 | ✅ 全部通过 |

### 测试覆盖内容
- Cron 表达式解析: `@hourly`, `@daily`, `@weekly` 别名
- 标准 5 字段 cron: `*/5 * * * *`, `0 9 * * 1-5`
- `matchCronField`: 通配符、步进、范围、列表、精确值、无效输入
- 边界条件: 无效表达式、字段数不匹配

---

## Python 测试详情 (38 tests)

| 文件 | 用例数 | 状态 |
|------|--------|------|
| `anomaly/test_detector.py` | 13 | ✅ 全部通过 |
| `llm/test_summary_engine.py` | 13 | ✅ 全部通过 |
| `rca/test_engine.py` | 12 | ✅ 全部通过 |

### Anomaly Detector (13 tests)
- 无数据/warmup 返回 None
- 尖峰/下降异常检测
- 静态阈值 (>, <) 优先级
- 严重程度分级 (critical/warning/info)
- 标签一致性、窗口大小

### LLM Summary Engine (13 tests)
- 正常指标 → normal 状态
- CPU > 80 → warning
- 内存 > 95 → critical
- 磁盘 > 90 → warning + 建议
- 错误率 > 5 → critical
- 日志错误数判断
- 事件升级逻辑
- 状态只升不降

### RCA Engine (12 tests)
- 时间序列添加/合并
- 因果图发现（指标不足/数据不足）
- 相关性发现（强相关/弱相关）
- 滞后估计
- 根因分析（有图/无图）
- 图结构输出

---

## 测试基础设施

### Frontend
- **框架**: vitest + @testing-library/react + jsdom
- **配置**: `web/vitest.config.ts`
- **运行**: `cd web && npm test`

### Go
- **框架**: 标准库 testing + testify
- **运行**: `go test ./internal/service/... -v`

### Python
- **框架**: pytest
- **配置**: `ai/pytest.ini`
- **运行**: `cd ai && python3 -m pytest . -v`

---

## 已知问题 (旧测试)

`tests/` 目录下的旧测试存在 9 个预存失败，原因：
1. mock 接口与当前生产代码不匹配
2. 测试文件未与源码共定位
3. RiverDetector mock 实现与实际 detector 行为不一致

这些旧测试不在本次测试范围内，后续可迁移或重构。
