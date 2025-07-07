#!/bin/bash

echo "========================================"
echo "SOCKS5 代理服务器测试脚本"
echo "========================================"

echo
echo "1. 编译测试程序..."
go build test_socks5.go

if [ $? -ne 0 ]; then
    echo "❌ 编译失败！"
    exit 1
fi

echo "✅ 编译成功！"

echo
echo "2. 运行测试..."
echo
./test_socks5

echo
echo "========================================"
echo "测试完成！"
echo "========================================" 