# SOCKS5 代理服务器快速开始指南

## 环境要求

- Go 1.21 或更高版本
- Windows/Linux/macOS 操作系统

## 快速安装

### 1. 克隆项目

```bash
git clone <repository-url>
cd go-socket5
```

### 2. 编译项目

```bash
go build
```

### 3. 运行服务器

```bash
./go-socket5
```

服务器将在 `0.0.0.0:1088` 上启动，默认用户名和密码都是 `test`。

## 基本使用

### 服务器配置

默认配置：

- 监听地址：`0.0.0.0`
- 监听端口：`1088`
- 用户名：`test`
- 密码：`test`
- 认证方法：无认证 + 用户名密码认证

### 客户端使用示例

#### 1. 基本 TCP 代理

```go
package main

import (
    "fmt"
    "net"
    "go-socket5/socket5"
)

func main() {
    // 创建客户端
    client := &socks5.Client{
        Host:     "127.0.0.1",
        Port:     1088,
        UserName: "test",
        Password: "test",
    }

    // 建立代理连接
    conn, err := client.TcpProxy("example.com", 80)
    if err != nil {
        fmt.Printf("连接失败: %v\n", err)
        return
    }
    defer conn.Close()

    // 发送HTTP请求
    conn.Write([]byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"))

    // 读取响应
    buffer := make([]byte, 1024)
    n, err := conn.Read(buffer)
    if err != nil {
        fmt.Printf("读取失败: %v\n", err)
        return
    }

    fmt.Printf("响应: %s\n", string(buffer[:n]))
}
```

#### 2. HTTP 代理客户端

```go
package main

import (
    "fmt"
    "io"
    "net/http"
    "go-socket5/socket5"
)

func main() {
    // 创建客户端
    client := &socks5.Client{
        Host:     "127.0.0.1",
        Port:     1088,
        UserName: "test",
        Password: "test",
    }

    // 获取HTTP代理客户端
    httpClient := client.GetHttpProxyClient()

    // 发送HTTP请求
    resp, err := httpClient.Get("http://httpbin.org/ip")
    if err != nil {
        fmt.Printf("请求失败: %v\n", err)
        return
    }
    defer resp.Body.Close()

    // 读取响应
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        fmt.Printf("读取响应失败: %v\n", err)
        return
    }

    fmt.Printf("响应: %s\n", string(body))
}
```

#### 3. UDP 代理

```go
package main

import (
    "fmt"
    "net"
    "go-socket5/socket5"
)

func main() {
    // 创建客户端
    client := &socks5.Client{
        Host:     "127.0.0.1",
        Port:     1088,
        UserName: "test",
        Password: "test",
    }

    // 建立UDP代理
    udpProxy, err := client.UdpProxy("8.8.8.8", 53)
    if err != nil {
        fmt.Printf("UDP代理失败: %v\n", err)
        return
    }
    defer udpProxy.Conn.Close()

    // 发送DNS查询
    query := []byte{0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x03, 0x63, 0x6f, 0x6d, 0x00, 0x00, 0x01, 0x00, 0x01}
    _, err = udpProxy.Conn.Write(query)
    if err != nil {
        fmt.Printf("发送DNS查询失败: %v\n", err)
        return
    }

    // 读取响应
    buffer := make([]byte, 512)
    n, err := udpProxy.Conn.Read(buffer)
    if err != nil {
        fmt.Printf("读取DNS响应失败: %v\n", err)
        return
    }

    fmt.Printf("DNS响应长度: %d\n", n)
}
```

## 测试连接

### 使用 curl 测试

```bash
# 使用SOCKS5代理
curl --socks5 127.0.0.1:1088 -u test:test http://httpbin.org/ip
```

### 使用 telnet 测试

```bash
# 连接到代理服务器
telnet 127.0.0.1 1088
```

### 使用 nc 测试

```bash
# 连接到代理服务器
nc 127.0.0.1 1088
```

## 配置选项

### 修改服务器配置

编辑 `main.go` 文件：

```go
const (
    port     = 1088        // 修改端口
    host     = "0.0.0.0"  // 修改监听地址
    userName = "test"      // 修改用户名
    password = "test"      // 修改密码
)
```

### 添加更多用户

```go
// 创建用户映射
userMap := map[string]string{
    "test":  "test",
    "admin": "admin123",
    "user1": "password1",
}
```

### 配置认证方法

```go
// 只支持无认证
AuthList: []uint8{socks5.NoAuthenticationRequired}

// 只支持用户名密码认证
AuthList: []uint8{socks5.AccountPasswordAuthentication}

// 支持两种认证方法
AuthList: []uint8{socks5.NoAuthenticationRequired, socks5.AccountPasswordAuthentication}
```

## 常见问题

### 1. 连接被拒绝

**问题：** 无法连接到代理服务器

**解决方案：**

- 检查服务器是否正在运行
- 确认端口是否正确
- 检查防火墙设置
- 验证监听地址配置

### 2. 认证失败

**问题：** 用户名或密码错误

**解决方案：**

- 确认用户名和密码正确
- 检查认证方法配置
- 查看服务器日志

### 3. 目标连接失败

**问题：** 无法连接到目标服务器

**解决方案：**

- 检查目标地址和端口
- 确认网络连接正常
- 验证目标服务器可访问

### 4. 数据传输问题

**问题：** 数据传输中断或错误

**解决方案：**

- 检查网络稳定性
- 确认代理服务器资源充足
- 查看错误日志

## 性能调优

### 1. 增加并发连接数

```go
// 在 server.go 中调整缓冲区大小
bytes := make([]byte, 4096)  // 增加缓冲区大小
```

### 2. 优化内存使用

```go
// 使用对象池减少内存分配
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 1024)
    },
}
```

### 3. 添加连接超时

```go
// 设置连接超时
conn.SetDeadline(time.Now().Add(time.Second * 30))
```

## 监控和日志

### 1. 添加日志记录

```go
import "log"

// 记录连接信息
log.Printf("新连接: %s", conn.RemoteAddr().String())

// 记录错误信息
log.Printf("连接错误: %v", err)
```

### 2. 性能监控

```go
// 统计连接数
var activeConnections int64

func (s *Server) newConn(conn net.Conn) {
    atomic.AddInt64(&activeConnections, 1)
    defer atomic.AddInt64(&activeConnections, -1)
    // ... 处理连接
}
```

## 安全建议

### 1. 使用强密码

```go
// 使用强密码
userMap := map[string]string{
    "admin": "StrongPassword123!",
}
```

### 2. 限制访问

```go
// 添加IP白名单
whitelist := []string{"192.168.1.0/24", "10.0.0.0/8"}
```

### 3. 启用 TLS

```go
// 使用TLS加密连接
cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
if err != nil {
    log.Fatal(err)
}

config := &tls.Config{Certificates: []tls.Certificate{cert}}
listener, err := tls.Listen("tcp", ":1088", config)
```

## 部署建议

### 1. 生产环境配置

```bash
# 使用systemd服务
sudo systemctl enable socks5-proxy
sudo systemctl start socks5-proxy
```

### 2. 反向代理

```nginx
# Nginx配置
server {
    listen 80;
    server_name proxy.example.com;

    location / {
        proxy_pass http://127.0.0.1:1088;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### 3. 负载均衡

```go
// 多服务器负载均衡
servers := []string{
    "server1:1088",
    "server2:1088",
    "server3:1088",
}
```

## 故障排除

### 1. 检查服务状态

```bash
# 检查端口是否监听
netstat -tlnp | grep 1088

# 检查进程
ps aux | grep go-socket5
```

### 2. 查看日志

```bash
# 查看系统日志
journalctl -u socks5-proxy

# 查看应用日志
tail -f socks5.log
```

### 3. 网络诊断

```bash
# 测试端口连通性
telnet 127.0.0.1 1088

# 抓包分析
tcpdump -i any port 1088
```

## 下一步

1. **阅读完整文档：** 查看 `README.md` 和 `TECHNICAL_DETAILS.md`
2. **运行测试：** 使用提供的测试用例验证功能
3. **自定义配置：** 根据需求调整服务器配置
4. **监控部署：** 在生产环境中部署和监控
5. **贡献代码：** 参与项目开发和改进

## 获取帮助

- **文档：** 查看项目文档
- **Issues：** 在 GitHub 上提交问题
- **讨论：** 参与社区讨论
- **邮件：** 联系项目维护者

---

**注意：** 本指南提供了基本的使用方法，更多高级功能请参考完整文档。
