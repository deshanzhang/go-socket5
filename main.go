package main

import (
	"fmt"
	"log"
	"time"

	"go-socket5/server"
	socks5 "go-socket5/socket5"
)

func main() {
	// 加载配置
	cfg, err := server.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 启动SOCKS5服务器
	socks5.ServerClient = &socks5.Server{
		Config: socks5.Config{
			Host:      cfg.Socks5.Host,
			Port:      uint16(cfg.Socks5.Port),
			BlackList: cfg.Socks5.BlackList,
			AuthList:  socks5.ToUint8Slice(cfg.Socks5.AuthList),
		},
		UserMap: map[string]string{
			cfg.Socks5.User: cfg.Socks5.Password,
		},
	}

	// 启动SOCKS5服务器
	go func() {
		log.Printf("启动SOCKS5服务器 %s:%d", cfg.Socks5.Host, cfg.Socks5.Port)
		err := socks5.ServerClient.Start()
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
	fmt.Println("💡 服务将持续运行，按 Ctrl+C 退出")

	select {}
}
