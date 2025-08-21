package main

import (
	"log"

	"github.com/google/uuid"

	"go-socket5/server"
	socks5 "go-socket5/socket5"
	"go-socket5/util"
)

func main() {
	// 加载配置
	cfg, err := server.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	//本地开启tcp信息生产token和唯一密码给用户使用 开放端口随机端口生成
	localAddr := uuid.New()
	localPasswd := uuid.New()
	localPort := util.RandomNumber(2000, 65530, 65535)

	// 上报数据交换鉴权信息
	//Host := "127.0.0.1"

	// 启动SOCKS5服务器
	socks5Server := &socks5.Server{
		Config: socks5.Config{
			Host:     cfg.Listen,
			Port:     uint16(localPort),
			AuthList: socks5.ToUint8Slice(cfg.AuthList),
		},
		UserMap: map[string]string{
			localAddr.String(): localPasswd.String(),
		},
	}
	log.Println("本地账户:", localAddr.String())
	log.Println("本地密钥:", localPasswd.String())

	// 启动SOCKS5服务器
	go func() {
		log.Printf("启动SOCKS5服务器 %s:%d", cfg.Listen, localPort)
		err := socks5Server.Start()
		if err != nil {
			log.Printf("SOCKS5服务器启动失败: %v", err)
		}
	}()

	select {}
}
