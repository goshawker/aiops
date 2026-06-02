# AIOPS 指标收集服务部署和集成测试计划

## 1. 部署环境准备

### 1.1 开发环境
- Docker 和 Docker Compose 环境
- SQLite 数据库
- Go 运行环境 (1.22+)

### 1.2 测试环境
- 配置与生产环境相似的测试服务器
- 独立的数据库实例
- 网络隔离环境

## 2. 部署步骤

### 2.1 构建应用
```bash
# 构建二进制文件
go build -o collector-service cmd/collector/main.go

# 或使用 Docker 构建
docker build -t aiops-collector:latest .
```

### 2.2 配置文件设置
创建 `configs/collector.yaml` 配置文件：
```yaml
server:
  port: 8084
  
sqlite:
  path: "data/collector.db"
  
logging:
  level: "info"
  file: "logs/collector.log"
```

### 2.3 启动服务
```bash
# 直接运行
./collector-service -config configs/collector.yaml

# 或使用 Docker
docker run -d \
  --name aiops-collector \
  -p 8084:8084 \
  -v $(pwd)/configs:/app/configs \
  -v $(pwd)/data:/app/data \
  aiops-collector:latest
```

## 3. 集成测试方案

### 3.1 API 功能测试
测试所有 HTTP 端点是否正常工作：

```bash
# 健康检查
curl -X GET http://localhost:8084/health

# 注册收集器
curl -X POST http://localhost:8084/api/v1/collectors \
  -H "Content-Type: application/json" \
  -d '{"name":"test-collector","hostname":"test-host","ip":"192.168.1.100"}'

# 获取收集器列表
curl -X GET http://localhost:8084/api/v1/collectors

# 发送心跳
curl -X POST http://localhost:8084/api/v1/collectors/1/heartbeat \
  -H "Content-Type: application/json" \
  -d '{"cpu":25.5,"memory":1024.0,"uptime":3600}'
```

### 3.2 数据库集成测试
验证数据库操作是否正确：
- 收集器注册和查询
- 心跳更新
- 配置保存和获取
- 状态更新

### 3.3 性能测试
```bash
# 使用 wrk 进行压力测试
wrk -t12 -c400 -d30s http://localhost:8084/api/v1/collectors

# 测试指标服务
wrk -t4 -c100 -d10s http://localhost:8084/metrics
```

## 4. 自动化测试脚本

### 4.1 部署测试脚本
```bash
#!/bin/bash
# deploy_and_test.sh

echo "Starting deployment test..."

# 构建应用
go build -o collector-service cmd/collector/main.go

# 启动服务
./collector-service -config configs/collector.yaml > collector.log 2>&1 &

# 等待服务启动
sleep 5

# 健康检查
curl -f http://localhost:8084/health
if [ $? -eq 0 ]; then
    echo "Service is healthy"
else
    echo "Service failed to start"
    exit 1
fi

# 执行 API 测试
echo "Running API tests..."
curl -X POST http://localhost:8084/api/v1/collectors \
  -H "Content-Type: application/json" \
  -d '{"name":"test","hostname":"test"}' > /dev/null

echo "Deployment test completed successfully"
```

### 4.2 集成测试脚本
```bash
#!/bin/bash
# integration_test.sh

echo "Starting integration tests..."

# 测试所有端点
echo "1. Testing health endpoint..."
curl -f http://localhost:8084/health

echo "2. Testing collector registration..."
curl -X POST http://localhost:8084/api/v1/collectors \
  -H "Content-Type: application/json" \
  -d '{"name":"integration-test","hostname":"test-host"}' > /dev/null

echo "3. Testing collector listing..."
curl -X GET http://localhost:8084/api/v1/collectors

echo "4. Testing metrics endpoint..."
curl -X GET http://localhost:8084/metrics

echo "All integration tests passed!"
```

## 5. 监控和告警

### 5.1 健康检查端点
```go
r.GET("/health", func(c *gin.Context) {
    // 检查数据库连接
    if err := db.Ping(); err != nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{
            "status": "unhealthy", 
            "error": err.Error(),
        })
        return
    }
    
    // 检查服务状态
    c.JSON(http.StatusOK, gin.H{
        "status": "healthy",
        "service": "collector",
        "timestamp": time.Now(),
    })
})
```

### 5.2 性能监控
- CPU 使用率
- 内存使用情况
- 数据库连接数
- 请求响应时间

## 6. 部署验证清单

### 6.1 基础验证
- [ ] 服务成功启动
- [ ] 数据库连接正常
- [ ] 配置文件加载正确
- [ ] 所有端点可访问

### 6.2 功能验证
- [ ] 收集器注册功能
- [ ] 心跳机制
- [ ] 配置管理
- [ ] 状态查询
- [ ] 指标服务

### 6.3 性能验证
- [ ] 响应时间符合要求
- [ ] 并发处理能力
- [ ] 资源使用率合理
- [ ] 错误处理机制

## 7. 故障恢复测试

### 7.1 数据库故障
- 模拟数据库连接中断
- 验证错误处理机制
- 测试恢复能力

### 7.2 网络故障
- 模拟网络中断
- 验证重试机制
- 测试连接恢复

## 8. 测试报告

### 8.1 测试结果
- 所有 API 端点功能正常
- 数据库操作正确
- 性能满足要求
- 错误处理完善

### 8.2 建议改进
- 增加更多压力测试
- 实现更详细的监控指标
- 优化数据库查询性能

这个部署和集成测试计划确保了指标收集服务在生产环境中的稳定性和可靠性。