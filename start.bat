@echo off
echo 启动SOCKS5代理服务器...
echo.
echo 配置信息:
echo - SOCKS5代理: 0.0.0.0:1080
echo - HTTP管理界面: http://0.0.0.0:8080
echo - 用户名: test
echo - 密码: test
echo.
echo 按任意键启动服务器...
pause >nul

go run main.go 