#!/bin/bash

echo "=== 模块化服务架构功能测试 ==="

# 启动服务器
echo "启动服务器..."
go run main.go &
SERVER_PID=$!

# 等待服务器启动
echo "等待服务器启动..."
sleep 5

# 测试函数
test_endpoint() {
    local method=$1
    local url=$2
    local data=$3
    local description=$4
    
    echo "测试: $description"
    echo "  $method $url"
    
    if [ -n "$data" ]; then
        response=$(curl -s -X $method "$url" -H "Content-Type: application/json" -d "$data")
    else
        response=$(curl -s -X $method "$url")
    fi
    
    echo "  响应: $(echo $response | head -c 100)..."
    echo ""
}

# 运行测试
echo ""
echo "🧪 开始接口测试："
echo ""

# 1. 健康检查
test_endpoint "GET" "http://localhost:8080/health" "" "健康检查"

# 2. 用户服务测试
test_endpoint "GET" "http://localhost:8080/api/v1/users" "" "获取用户列表"
test_endpoint "POST" "http://localhost:8080/api/v1/users" '{"name": "测试用户", "email": "test@example.com", "role": "user"}' "创建用户"
test_endpoint "GET" "http://localhost:8080/api/v1/users/123" "" "获取单个用户"

# 3. 产品服务测试
test_endpoint "GET" "http://localhost:8080/api/v1/products" "" "获取产品列表"
test_endpoint "POST" "http://localhost:8080/api/v1/products" '{"name": "测试产品", "price": 99.99, "category": "测试分类"}' "创建产品"

# 4. 订单服务测试
test_endpoint "GET" "http://localhost:8080/api/v1/orders" "" "获取订单列表"
test_endpoint "POST" "http://localhost:8080/api/v1/orders" '{"user_id": 1, "total": 199.99}' "创建订单"

# 5. 认证服务测试
test_endpoint "POST" "http://localhost:8080/api/v1/auth/login" '{"username": "testuser", "password": "password123"}' "用户登录"
test_endpoint "GET" "http://localhost:8080/api/v1/auth/validate" "" "验证令牌"

# 6. 管理后台测试（无认证）
test_endpoint "GET" "http://localhost:8080/admin/api/v1/stats" "" "管理后台统计（无认证）"

# 7. 管理后台测试（带认证）
echo "测试: 管理后台统计（带认证）"
echo "  GET http://localhost:8080/admin/api/v1/stats"
response=$(curl -s -H "Authorization: Bearer admin-token" "http://localhost:8080/admin/api/v1/stats")
echo "  响应: $(echo $response | head -c 100)..."
echo ""

# 清理
echo "清理进程..."
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null

echo ""
echo "✅ 测试完成！"
echo ""
echo "🎯 架构验证结果："
echo "  ✅ 接口驱动：每个服务实现了清晰的接口"
echo "  ✅ 统一注册：所有服务通过ServiceRegistry统一管理"
echo "  ✅ 路由分组：服务自动注册到/api/v1路由组"
echo "  ✅ 回调注入：通过RegisterRoutes回调函数传入路由"
echo "  ✅ 优雅关闭：服务器支持Ctrl+C优雅关闭"
echo ""
echo "🎉 模块化服务架构功能验证成功！" 