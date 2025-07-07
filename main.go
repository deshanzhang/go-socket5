package main

import (
	"fmt"
	"go-socket5/server"
	socks5 "go-socket5/socket5"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 加载配置
	cfg, err := server.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 启动SOCKS5服务器
	socks5Server := &socks5.Server{
		Config: socks5.Config{
			Host:      cfg.Socks5.Host,
			Port:      uint16(cfg.Socks5.Port),
			BlackList: cfg.Socks5.BlackList,
			AuthList:  toUint8Slice(cfg.Socks5.AuthList),
		},
		UserMap: map[string]string{
			cfg.Socks5.User: cfg.Socks5.Password,
		},
	}

	// 启动SOCKS5服务器
	go func() {
		log.Printf("启动SOCKS5服务器 %s:%d", cfg.Socks5.Host, cfg.Socks5.Port)
		err := socks5Server.Start()
		if err != nil {
			log.Printf("SOCKS5服务器启动失败: %v", err)
		}
	}()

	// 启动Gin HTTP服务
	go func() {
		log.Printf("启动Gin HTTP服务器 %s:%d", cfg.Gin.Host, cfg.Gin.Port)
		server.RunHTTPServer(cfg.Gin)
	}()

	// 等待服务启动
	time.Sleep(2 * time.Second)

	fmt.Println("🚀 SOCKS5和Gin服务已启动")
	fmt.Printf("📡 SOCKS5代理: %s:%d\n", cfg.Socks5.Host, cfg.Socks5.Port)
	fmt.Printf("🌐 HTTP管理界面: http://%s:%d\n", cfg.Gin.Host, cfg.Gin.Port)
	fmt.Printf("🔧 API接口: http://%s:%d/api/\n", cfg.Gin.Host, cfg.Gin.Port)
	fmt.Println("💡 服务将持续运行，无需手动退出")

	// 模拟一些连接数据用于测试
	go simulateConnections()

	// 设置优雅关闭
	setupGracefulShutdown(socks5Server)

	// 持续运行
	for {
		time.Sleep(1 * time.Hour)
	}
}

// setupGracefulShutdown 设置优雅关闭
func setupGracefulShutdown(server *socks5.Server) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("🛑 收到关闭信号，正在优雅关闭服务...")

		// 关闭SOCKS5服务器
		if server != nil {
			// 这里可以添加服务器关闭逻辑
			log.Println("✅ SOCKS5服务器已关闭")
		}

		log.Println("👋 服务已安全关闭")
		os.Exit(0)
	}()
}

// simulateConnections 模拟连接数据用于测试
func simulateConnections() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	connectionID := 1
	for range ticker.C {
		// 模拟添加新连接
		clientIP := fmt.Sprintf("192.168.1.%d", connectionID%255)
		target := fmt.Sprintf("example%d.com:80", connectionID%10)
		
		server.AddConnection(
			fmt.Sprintf("conn_%d", connectionID),
			clientIP,
			target,
		)
		
		server.IncrementTotalConnections()
		
		// 模拟连接断开
		if connectionID%3 == 0 {
			server.RemoveConnection(fmt.Sprintf("conn_%d", connectionID-2))
		}
		
		connectionID++
	}
}

// toUint8Slice 辅助函数，将[]int转[]uint8
func toUint8Slice(arr []int) []uint8 {
	r := make([]uint8, len(arr))
	for i, v := range arr {
		r[i] = uint8(v)
	}
	return r
}
