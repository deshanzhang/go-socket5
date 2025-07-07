@echo off
echo ========================================
echo SOCKS5 代理服务器测试脚本
echo ========================================

echo.
echo 1. 编译测试程序...
go build test_socks5.go

if %errorlevel% neq 0 (
    echo ❌ 编译失败！
    pause
    exit /b 1
)

echo ✅ 编译成功！

echo.
echo 2. 运行测试...
echo.
test_socks5.exe

echo.
echo ========================================
echo 测试完成！
echo ========================================
pause 