#!/bin/bash
# deploy_and_test.sh

echo "Starting deployment test..."

# Create necessary directories
mkdir -p /Users/LB/Documents/AIOPS/test-deployment/configs
mkdir -p /Users/LB/Documents/AIOPS/test-deployment/data
mkdir -p /Users/LB/Documents/AIOPS/test-deployment/logs

# Create test configuration file
cat > /Users/LB/Documents/AIOPS/test-deployment/configs/collector.yaml << EOF
server:
  port: 8085

sqlite:
  path: "data/collector.db"

logging:
  level: "debug"
  file: "logs/collector.log"
EOF

# Build the application
echo "Building collector service..."
cd /Users/LB/Documents/AIOPS
go build -o test-deployment/collector-service cmd/collector/main.go

if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

echo "Build successful!"

# Start the service
echo "Starting collector service..."
cd /Users/LB/Documents/AIOPS/test-deployment
./collector-service -config configs/collector.yaml > collector.log 2>&1 &

# Get the process ID
PID=$!

# Wait for service to start
echo "Waiting for service to start..."
sleep 5

# Check if service is running
if ps -p $PID > /dev/null; then
    echo "Service started successfully with PID: $PID"
else
    echo "Service failed to start"
    cat collector.log
    exit 1
fi

# Test health endpoint
echo "Testing health endpoint..."
curl -f http://localhost:8085/health
if [ $? -eq 0 ]; then
    echo "Health check passed"
else
    echo "Health check failed"
    cat collector.log
    exit 1
fi

# Test collector registration
echo "Testing collector registration..."
curl -X POST http://localhost:8085/api/v1/collectors \
  -H "Content-Type: application/json" \
  -d '{"name":"test-collector","hostname":"test-host","ip":"192.168.1.100"}' > /dev/null

if [ $? -eq 0 ]; then
    echo "Collector registration test passed"
else
    echo "Collector registration test failed"
    exit 1
fi

# Test collector listing
echo "Testing collector listing..."
curl -X GET http://localhost:8085/api/v1/collectors

if [ $? -eq 0 ]; then
    echo "Collector listing test passed"
else
    echo "Collector listing test failed"
    exit 1
fi

# Test heartbeat
echo "Testing heartbeat..."
curl -X POST http://localhost:8085/api/v1/collectors/1/heartbeat \
  -H "Content-Type: application/json" \
  -d '{"cpu":25.5,"memory":1024.0,"uptime":3600}' > /dev/null

if [ $? -eq 0 ]; then
    echo "Heartbeat test passed"
else
    echo "Heartbeat test failed"
    exit 1
fi

# Test metrics endpoint
echo "Testing metrics endpoint..."
curl -X GET http://localhost:8085/metrics

if [ $? -eq 0 ]; then
    echo "Metrics endpoint test passed"
else
    echo "Metrics endpoint test failed"
    exit 1
fi

# Cleanup
echo "Cleaning up..."
kill $PID 2>/dev/null

echo "All tests passed successfully!"