#!/bin/bash

echo "=== 配置管理器环境变量前缀测试 ==="
echo

echo "1. 测试使用前缀（APP_NAME=myapp）"
echo "设置环境变量..."
export APP_NAME=myapp
export MYAPP_SERVER_HOST=localhost
export MYAPP_SERVER_PORT=8080
export MYAPP_DEBUG=true
export MYAPP_DATABASE_HOST=localhost
export MYAPP_DATABASE_PORT=3306
export MYAPP_DATABASE_NAME=myapp_db

echo "运行程序..."
go run main.go

echo
echo "----------------------------------------"
echo

echo "2. 测试不使用前缀（不设置 APP_NAME）"
echo "清除 APP_NAME 并设置无前缀环境变量..."
unset APP_NAME
export SERVER_HOST=127.0.0.1
export SERVER_PORT=9090
export DEBUG=false
export DATABASE_HOST=127.0.0.1
export DATABASE_PORT=5432
export DATABASE_NAME=test_db

echo "运行程序..."
go run main.go

echo
echo "=== 测试完成 ==="

# 清理环境变量
unset APP_NAME
unset MYAPP_SERVER_HOST MYAPP_SERVER_PORT MYAPP_DEBUG
unset MYAPP_DATABASE_HOST MYAPP_DATABASE_PORT MYAPP_DATABASE_NAME
unset SERVER_HOST SERVER_PORT DEBUG
unset DATABASE_HOST DATABASE_PORT DATABASE_NAME 