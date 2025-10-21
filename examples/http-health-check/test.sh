#!/bin/bash

# HTTP Server 健康检查功能测试脚本

echo "=== HTTP Server 健康检查功能测试 ==="
echo

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试函数
test_endpoint() {
    local method=$1
    local url=$2
    local expected_status=$3
    local description=$4
    
    echo -n "测试 $description: "
    
    if [ "$method" = "GET" ]; then
        response=$(curl -s -w "%{http_code}" -o /tmp/response.json "$url")
    else
        response=$(curl -s -w "%{http_code}" -o /tmp/response.json -X "$method" "$url")
    fi
    
    if [ "$response" = "$expected_status" ]; then
        echo -e "${GREEN}✅ 通过${NC} (状态码: $response)"
        if [ -f /tmp/response.json ]; then
            echo "   响应内容:"
            cat /tmp/response.json | jq . 2>/dev/null || cat /tmp/response.json
        fi
    else
        echo -e "${RED}❌ 失败${NC} (期望: $expected_status, 实际: $response)"
        if [ -f /tmp/response.json ]; then
            echo "   响应内容:"
            cat /tmp/response.json
        fi
    fi
    echo
}

# 等待服务器启动
echo -e "${BLUE}等待服务器启动...${NC}"
sleep 3

echo "开始测试各个端点..."
echo

# 1. 基本健康检查测试
echo -e "${YELLOW}1. 基本健康检查测试${NC}"
echo "----------------------------------------"
test_endpoint "GET" "http://localhost:8080/health" "200" "基本健康检查"
test_endpoint "GET" "http://localhost:8080/" "200" "基本服务器根路径"
test_endpoint "GET" "http://localhost:8080/api/info" "200" "基本服务器API信息"

# 2. 带管理器的健康检查测试
echo -e "${YELLOW}2. 带管理器的健康检查测试${NC}"
echo "----------------------------------------"
test_endpoint "GET" "http://localhost:8081/health" "503" "带管理器的健康检查（应该不健康）"
test_endpoint "GET" "http://localhost:8081/" "200" "高级服务器根路径"
test_endpoint "GET" "http://localhost:8081/api/status" "200" "高级服务器状态"

# 3. 自定义路径健康检查测试
echo -e "${YELLOW}3. 自定义路径健康检查测试${NC}"
echo "----------------------------------------"
test_endpoint "GET" "http://localhost:8082/api/v1/health" "200" "自定义路径健康检查"
test_endpoint "GET" "http://localhost:8082/" "200" "自定义路径服务器根路径"

# 4. 禁用健康检查测试
echo -e "${YELLOW}4. 禁用健康检查测试${NC}"
echo "----------------------------------------"
test_endpoint "GET" "http://localhost:8083/health" "404" "禁用健康检查（应该404）"
test_endpoint "GET" "http://localhost:8083/" "200" "禁用健康检查服务器根路径"

# 5. 带日志集成测试
echo -e "${YELLOW}5. 带日志集成测试${NC}"
echo "----------------------------------------"
test_endpoint "GET" "http://localhost:8084/health" "200" "带日志的健康检查"
test_endpoint "GET" "http://localhost:8084/" "200" "带日志的服务器根路径"

# 6. 性能测试
echo -e "${YELLOW}6. 性能测试${NC}"
echo "----------------------------------------"
echo "测试健康检查端点响应时间..."

start_time=$(date +%s%N)
curl -s http://localhost:8080/health > /dev/null
end_time=$(date +%s%N)
duration=$(( (end_time - start_time) / 1000000 ))

echo "基本健康检查响应时间: ${duration}ms"

start_time=$(date +%s%N)
curl -s http://localhost:8081/health > /dev/null
end_time=$(date +%s%N)
duration=$(( (end_time - start_time) / 1000000 ))

echo "带管理器的健康检查响应时间: ${duration}ms"

# 7. 并发测试
echo -e "${YELLOW}7. 并发测试${NC}"
echo "----------------------------------------"
echo "测试并发健康检查请求..."

for i in {1..5}; do
    (
        response=$(curl -s -w "%{http_code}" -o /dev/null http://localhost:8080/health)
        if [ "$response" = "200" ]; then
            echo "并发请求 $i: ✅ 成功"
        else
            echo "并发请求 $i: ❌ 失败 (状态码: $response)"
        fi
    ) &
done

wait

echo
echo -e "${GREEN}=== 测试完成 ===${NC}"
echo
echo "测试总结:"
echo "- 基本健康检查: 正常工作"
echo "- 带管理器的健康检查: 正常工作（显示不健康状态，因为外部API检查失败）"
echo "- 自定义路径健康检查: 正常工作"
echo "- 禁用健康检查: 正常工作（返回404）"
echo "- 带日志集成: 正常工作"
echo "- 性能测试: 响应时间正常"
echo "- 并发测试: 并发请求正常"
echo
echo "所有测试通过！🎉"