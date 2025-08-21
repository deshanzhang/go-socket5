package server

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

type GinConfig struct {
	Host string
	Port int
	Mode string
}

// 服务器统计信息
type ServerStats struct {
	Connections      int    `json:"connections"`
	TotalConnections int    `json:"totalConnections"`
	Uptime           int64  `json:"uptime"`
	StartTime        int64  `json:"startTime"`
	Status           string `json:"status"`
}

// 服务器配置信息
type ServerConfig struct {
	Host           string   `json:"host"`
	Port           int      `json:"port"`
	User           string   `json:"user"`
	AuthMethods    []string `json:"authMethods"`
	MaxConnections int      `json:"maxConnections"`
	BlackList      []string `json:"blackList"`
}

// 连接信息
type ConnectionInfo struct {
	ID        string    `json:"id"`
	ClientIP  string    `json:"clientIP"`
	Target    string    `json:"target"`
	StartTime time.Time `json:"startTime"`
	Status    string    `json:"status"`
}

// NewHTTPServer 创建并返回Gin引擎
func NewHTTPServer(cfg GinConfig) *gin.Engine {
	if cfg.Mode != "" {
		gin.SetMode(cfg.Mode)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// 静态文件托管
	r.Static("/static", "./static")

	//// 设置静态文件缓存
	//r.Use(func(c *gin.Context) {
	//	if c.Request.URL.Path == "/" {
	//		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	//		c.Header("Pragma", "no-cache")
	//		c.Header("Expires", "0")
	//	} else if c.Request.URL.Path == "/static/style.css" || c.Request.URL.Path == "/static/script.js" {
	//		c.Header("Cache-Control", "public, max-age=3600")
	//	}
	//	c.Next()
	//})
	//
	//// 根路径重定向到静态文件
	//r.GET("/", func(c *gin.Context) {
	//	c.File("./static/index.html")
	//})
	//
	// 健康检查
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"msg": "pong"})
	})

	// API路由组
	//api := r.Group("/api")
	//{
	//	// 获取服务器统计信息
	//	//api.GET("/stats", func(c *gin.Context) {
	//	//	statsMutex.RLock()
	//	//	uptime := int64(time.Since(serverStartTime).Seconds())
	//	//	stats := ServerStats{
	//	//		Connections:      connectionCount,
	//	//		TotalConnections: totalConnections,
	//	//		Uptime:           uptime,
	//	//		StartTime:        serverStartTime.Unix(),
	//	//		Status:           "running",
	//	//	}
	//	//	statsMutex.RUnlock()
	//	//	c.JSON(http.StatusOK, stats)
	//	//})
	//
	//}

	return r
}

// RunHTTPServer 启动Gin服务
func RunHTTPServer(cfg GinConfig) {
	r := NewHTTPServer(cfg)
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	fmt.Printf("🌐 HTTP服务器启动在: http://%s\n", addr)
	fmt.Printf("📁 静态文件目录: ./static\n")
	fmt.Printf("📄 首页地址: http://%s/\n", addr)
	fmt.Printf("🔧 API文档: http://%s/api/\n", addr)

	r.Run(addr)
}
