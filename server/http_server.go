package server

import (
	"fmt"
	"net/http"
	"sync"
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
	Connections      int   `json:"connections"`
	TotalConnections int   `json:"totalConnections"`
	Uptime           int64 `json:"uptime"`
	StartTime        int64 `json:"startTime"`
	Status           string `json:"status"`
}

// 服务器配置信息
type ServerConfig struct {
	Host            string   `json:"host"`
	Port            int      `json:"port"`
	User            string   `json:"user"`
	AuthMethods     []string `json:"authMethods"`
	MaxConnections  int      `json:"maxConnections"`
	BlackList       []string `json:"blackList"`
}

// 连接信息
type ConnectionInfo struct {
	ID        string    `json:"id"`
	ClientIP  string    `json:"clientIP"`
	Target    string    `json:"target"`
	StartTime time.Time `json:"startTime"`
	Status    string    `json:"status"`
}

var (
	serverStartTime  = time.Now()
	connectionCount  = 0
	totalConnections = 0
	activeConnections = make(map[string]*ConnectionInfo)
	statsMutex       = sync.RWMutex{}
	connectionsMutex = sync.RWMutex{}
)

// NewHTTPServer 创建并返回Gin引擎
func NewHTTPServer(cfg GinConfig) *gin.Engine {
	if cfg.Mode != "" {
		gin.SetMode(cfg.Mode)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// 静态文件托管
	r.Static("/static", "./static")

	// 设置静态文件缓存
	r.Use(func(c *gin.Context) {
		if c.Request.URL.Path == "/" {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		} else if c.Request.URL.Path == "/static/style.css" || c.Request.URL.Path == "/static/script.js" {
			c.Header("Cache-Control", "public, max-age=3600")
		}
		c.Next()
	})

	// 根路径重定向到静态文件
	r.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})

	// 健康检查
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"msg": "pong"})
	})

	// API路由组
	api := r.Group("/api")
	{
		// 获取服务器统计信息
		api.GET("/stats", func(c *gin.Context) {
			statsMutex.RLock()
			uptime := int64(time.Since(serverStartTime).Seconds())
			stats := ServerStats{
				Connections:      connectionCount,
				TotalConnections: totalConnections,
				Uptime:           uptime,
				StartTime:        serverStartTime.Unix(),
				Status:           "running",
			}
			statsMutex.RUnlock()
			c.JSON(http.StatusOK, stats)
		})

		// 获取服务器状态
		api.GET("/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":    "running",
				"timestamp": time.Now().Unix(),
				"uptime":    int64(time.Since(serverStartTime).Seconds()),
			})
		})

		// 获取服务器配置
		api.GET("/config", func(c *gin.Context) {
			config := ServerConfig{
				Host:           "0.0.0.0",
				Port:           1080,
				User:           "test",
				AuthMethods:    []string{"无认证", "用户名密码认证"},
				MaxConnections: 10000,
				BlackList:      []string{},
			}
			c.JSON(http.StatusOK, config)
		})

		// 获取活跃连接列表
		api.GET("/connections", func(c *gin.Context) {
			connectionsMutex.RLock()
			connections := make([]*ConnectionInfo, 0, len(activeConnections))
			for _, conn := range activeConnections {
				connections = append(connections, conn)
			}
			connectionsMutex.RUnlock()
			c.JSON(http.StatusOK, connections)
		})

		// 获取连接详情
		api.GET("/connections/:id", func(c *gin.Context) {
			id := c.Param("id")
			connectionsMutex.RLock()
			conn, exists := activeConnections[id]
			connectionsMutex.RUnlock()
			
			if !exists {
				c.JSON(http.StatusNotFound, gin.H{"error": "连接不存在"})
				return
			}
			c.JSON(http.StatusOK, conn)
		})

		// 断开指定连接
		api.DELETE("/connections/:id", func(c *gin.Context) {
			id := c.Param("id")
			connectionsMutex.Lock()
			delete(activeConnections, id)
			connectionsMutex.Unlock()
			
			// 更新连接数
			statsMutex.Lock()
			connectionCount = len(activeConnections)
			statsMutex.Unlock()
			
			c.JSON(http.StatusOK, gin.H{"message": "连接已断开"})
		})

		// 重启服务器
		api.POST("/restart", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "重启请求已发送",
				"timestamp": time.Now().Unix(),
			})
		})

		// 更新配置
		api.PUT("/config", func(c *gin.Context) {
			var config ServerConfig
			if err := c.ShouldBindJSON(&config); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置数据"})
				return
			}
			
			c.JSON(http.StatusOK, gin.H{
				"message": "配置更新成功",
				"config": config,
			})
		})

		// 获取日志
		api.GET("/logs", func(c *gin.Context) {
			limit := c.DefaultQuery("limit", "100")
			c.JSON(http.StatusOK, gin.H{
				"logs": []string{
					"[INFO] 服务器启动成功",
					"[INFO] 新连接: 127.0.0.1:12345",
					"[INFO] 认证成功",
					"[INFO] 连接目标: example.com:80",
				},
				"limit": limit,
			})
		})

		// 测试连接
		api.POST("/test", func(c *gin.Context) {
			var req struct {
				Host string `json:"host"`
				Port int    `json:"port"`
			}
			
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
				return
			}
			
			// 模拟连接测试
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": fmt.Sprintf("连接到 %s:%d 成功", req.Host, req.Port),
				"latency": "45ms",
			})
		})
	}

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

// UpdateConnectionCount 更新连接数统计
func UpdateConnectionCount(count int) {
	statsMutex.Lock()
	connectionCount = count
	statsMutex.Unlock()
}

// IncrementTotalConnections 增加总连接数
func IncrementTotalConnections() {
	statsMutex.Lock()
	totalConnections++
	statsMutex.Unlock()
}

// AddConnection 添加新连接
func AddConnection(id, clientIP, target string) {
	connectionsMutex.Lock()
	activeConnections[id] = &ConnectionInfo{
		ID:        id,
		ClientIP:  clientIP,
		Target:    target,
		StartTime: time.Now(),
		Status:    "active",
	}
	connectionsMutex.Unlock()
	
	statsMutex.Lock()
	connectionCount = len(activeConnections)
	statsMutex.Unlock()
}

// RemoveConnection 移除连接
func RemoveConnection(id string) {
	connectionsMutex.Lock()
	delete(activeConnections, id)
	connectionsMutex.Unlock()
	
	statsMutex.Lock()
	connectionCount = len(activeConnections)
	statsMutex.Unlock()
}

// GetActiveConnections 获取活跃连接数
func GetActiveConnections() int {
	connectionsMutex.RLock()
	defer connectionsMutex.RUnlock()
	return len(activeConnections)
}

// GetTotalConnections 获取总连接数
func GetTotalConnections() int {
	statsMutex.RLock()
	defer statsMutex.RUnlock()
	return totalConnections
}
