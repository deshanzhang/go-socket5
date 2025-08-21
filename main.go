package main

import (
	"fmt"
	"log"

	"github.com/go-resty/resty/v2"
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
	localAddr := uuid.New().String()
	localPasswd := uuid.New().String()
	localPort := util.RandomNumber(2000, 65530, 65535)

	// 启动SOCKS5服务器
	socks5Server := &socks5.Server{
		Config: socks5.Config{
			Host:     cfg.Listen,
			Port:     uint16(localPort),
			AuthList: socks5.ToUint8Slice(cfg.AuthList),
		},
		UserMap: map[string]string{
			localAddr: localPasswd,
		},
	}
	log.Println("本地账户:", localAddr)
	log.Println("本地密钥:", localPasswd)

	// 启动SOCKS5服务器
	go func() {
		log.Printf("启动SOCKS5服务器 %s:%d", cfg.Listen, localPort)
		err := socks5Server.Start()
		if err != nil {
			log.Printf("SOCKS5服务器启动失败: %v", err)
		}
	}()

	// 上报数据交换鉴权信息
	log.Printf("开始链接服务器进行上报数据，baseUrl:%s", cfg.HttpServer)
	ipv4, err := util.ThisIpv4Address()
	if err != nil {
		log.Fatalf("获取ip地址异常: %v", err)
	}
	in := socks5.Client{
		Host:     ipv4,
		Port:     uint16(localPort),
		UserName: localAddr,
		Password: localPasswd,
	}
	res, err := resty.New().R().EnableTrace().SetBody(in).Post(fmt.Sprintf("%s/auth", cfg.HttpServer))
	if err != nil {
		log.Fatalf("链接鉴权服务器错误: %v", err)
	}
	resp := socks5.Client{}
	_, err = util.ReResponse(res, &resp)
	if err != nil {
		log.Fatalf("解析鉴权服务器数据失败: %v", err)
	}
	util.ConsoleJson(resp)
	// 反向代理实现转发
	log.Printf("开始检查反向代理服务，baseUrl:%s", cfg.HttpServer)
	if err := socks5.TestConnection(resp, []string{"baidu.com", "bilibili.com"}); err != nil {
		log.Fatalf("检查代理服务器错误: %v", err)
	}

	select {}
}
