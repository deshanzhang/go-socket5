# SOCKS5 代理服务器

## 项目简介

这是一个完整的 SOCKS5 代理服务器实现，包含服务器端和客户端功能。SOCKS5 是一种网络代理协议，让你可以通过代理服务器访问网络资源。

**主要特性：**

- ✅ 完整的 SOCKS5 协议实现（CONNECT、BIND、UDP）
- ✅ 支持用户名密码认证
- ✅ 清晰的日志输出
- ✅ 详细的调试信息
- ✅ 完善的错误处理

## 项目结构

```
go-socket5/
├── main.go              # 主程序入口和测试用例
├── go.mod               # Go模块文件
├── README.md            # 项目文档
└── socket5/
    ├── constants.go     # 常量定义
    ├── server.go        # 服务器端实现
    ├── client.go        # 客户端实现
    └── utils.go         # 工具函数
```

## 核心功能

### 1. SOCKS5 协议支持

- **CONNECT 命令**: 建立 TCP 连接到目标服务器
- **BIND 命令**: 绑定端口等待连接
- **UDP 命令**: UDP 数据包转发
- **认证方式**: 无认证、用户名密码认证

### 2. 日志系统

项目使用统一的日志格式，方便调试和监控：

```go
const (
    LogPrefixServer = "[SOCKS5-SERVER]"  // 服务器日志前缀
    LogPrefixClient = "[SOCKS5-CLIENT]"  // 客户端日志前缀
    LogPrefixUtils  = "[SOCKS5-UTILS]"   // 工具函数日志前缀
)
```

**日志输出示例：**

```
[SOCKS5-SERVER] 启动服务器 0.0.0.0:1080
[SOCKS5-SERVER] 服务器启动成功，等待连接...
[SOCKS5-SERVER] 新连接: 127.0.0.1:60464
[SOCKS5-CLIENT] 开始认证，用户名: test, 密码: test
[SOCKS5-SERVER] 认证成功
[SOCKS5-CLIENT] 开始发送SOCKS5请求: cmd=1, host=www.baidu.com, port=80
[SOCKS5-UTILS] 解析地址: addrType=3, data=0e7777772e62616964752e636f6d0050
[SOCKS5-UTILS] 域名解析结果: www.baidu.com:80
```

## 核心文件说明

### 1. constants.go - 常量定义

定义了 SOCKS5 协议相关的所有常量：

```go
const (
    Version = 0x05                    // SOCKS5版本号
    NoAuthenticationRequired = 0x00    // 无认证
    AccountPasswordAuthentication = 0x02 // 用户名密码认证
    Connect = 0x01                    // CONNECT命令
    Bind = 0x02                       // BIND命令
    UDP = 0x03                        // UDP命令
    IPv4 = 0x01                       // IPv4地址类型
    Domain = 0x03                     // 域名地址类型
    IPv6 = 0x04                       // IPv6地址类型
    Zero = 0x00                       // 成功状态码
)
```

### 2. server.go - 服务器端实现

#### 核心结构体

```go
type Server struct {
    listen  net.Listener    // TCP监听器
    Config  Config          // 服务器配置
    UserMap map[string]string // 用户认证映射
}

type Config struct {
    Host      string   // 监听地址
    Port      uint16   // 监听端口
    BlackList []string // 黑名单
    AuthList  []uint8  // 支持的认证方法
}
```

#### 主要方法

1. **Start()** - 启动服务器

   - 创建 TCP 监听器
   - 接受客户端连接
   - 为每个连接启动 goroutine 处理

2. **newConn(conn net.Conn)** - 处理客户端连接

   - 执行认证流程
   - 解析客户端请求
   - 根据命令类型处理连接

3. **auth(conn net.Conn)** - 认证处理
   - 协商认证方法
   - 支持无认证和用户名密码认证
   - 验证用户凭据

#### SOCKS5 协议流程

1. **认证阶段**

   ```
   客户端 -> 服务器: [0x05, 0x02, 0x00, 0x02]  // 支持的认证方法
   服务器 -> 客户端: [0x05, 0x02]               // 选择认证方法
   客户端 -> 服务器: [0x01, 0x04, 't', 'e', 's', 't', 0x04, 't', 'e', 's', 't']  // 用户名密码
   服务器 -> 客户端: [0x01, 0x00]               // 认证结果
   ```

2. **请求阶段**
   ```
   客户端 -> 服务器: [0x05, 0x01, 0x00, 0x03, 0x0e, 'w', 'w', 'w', '.', 'b', 'a', 'i', 'd', 'u', '.', 'c', 'o', 'm', 0x00, 0x50]  // CONNECT请求
   服务器 -> 客户端: [0x05, 0x00, 0x00, 0x01, 192, 168, 1, 100, 0x00, 0x50] // 响应
   ```

### 3. client.go - 客户端实现

#### 核心结构体

```go
type Client struct {
    Host     string // 代理服务器地址
    Port     uint16 // 代理服务器端口
    UserName string // 用户名
    Password string // 密码
}

type UdpProxy struct {
    Conn    net.Conn     // TCP控制连接
    UdpConn *net.UDPConn // UDP数据连接
    Host    string       // 目标主机
    Port    uint16       // 目标端口
    client  *Client      // 客户端引用
}
```

#### 主要方法

1. **conn()** - 建立到代理服务器的连接
2. **auth(conn net.Conn)** - 执行认证流程
3. **requisition(conn net.Conn, host string, port uint16, cmd uint8)** - 发送代理请求
4. **TcpProxy(host string, port uint16)** - TCP 代理
5. **UdpProxy(host string, port uint16)** - UDP 代理
6. **GetHttpProxyClient()** - 获取 HTTP 代理客户端

### 4. utils.go - 工具函数

#### 地址解析函数

1. **addressResolutionFormByteArray(data []byte, addrType byte)** - 从字节数组解析地址

   - 支持 IPv4、IPv6、域名地址类型
   - 返回格式化的地址字符串
   - 包含详细的调试日志

2. **addressResolution(addr string)** - 解析地址字符串

   - 解析主机和端口
   - 返回地址类型、地址字节、端口
   - 正确处理域名格式（包含长度字节）

3. **ioCopy(dst net.Conn, src net.Conn)** - 连接间数据复制
   - 双向数据转发
   - 自动关闭连接

## 使用示例

### 1. 基本使用

```go
package main

import (
    "fmt"
    "log"
    "time"
    socks5 "go-socket5/socket5"
)

func main() {
    // 启动SOCKS5服务器
    server := &socks5.Server{
        Config: socks5.Config{
            Host:      "0.0.0.0",
            Port:      1080,
            BlackList: []string{},
            AuthList:  []uint8{socks5.AccountPasswordAuthentication},
        },
        UserMap: map[string]string{
            "test": "test",
        },
    }

    // 在后台启动服务器
    go func() {
        err := server.Start()
        if err != nil {
            log.Printf("服务器启动失败: %v", err)
        }
    }()

    // 等待服务器启动
    time.Sleep(2 * time.Second)

    // 创建客户端
    client := &socks5.Client{
        Host:     "127.0.0.1",
        Port:     1080,
        UserName: "test",
        Password: "test",
    }

    // TCP代理测试
    conn, err := client.TcpProxy("www.baidu.com", 80)
    if err != nil {
        log.Printf("TCP代理失败: %v", err)
    } else {
        fmt.Println("TCP代理成功")
        conn.Close()
    }

    // UDP代理测试
    udpProxy, err := client.UdpProxy("8.8.8.8", 53)
    if err != nil {
        log.Printf("UDP代理失败: %v", err)
    } else {
        fmt.Println("UDP代理成功")
        udpProxy.Close()
    }
}
```

### 2. HTTP 代理客户端

```go
// 获取配置了SOCKS5代理的HTTP客户端
httpClient := client.GetHttpProxyClient()

// 使用HTTP客户端发送请求
resp, err := httpClient.Get("http://www.example.com")
if err != nil {
    log.Printf("HTTP请求失败: %v", err)
} else {
    defer resp.Body.Close()
    fmt.Printf("HTTP状态码: %d\n", resp.StatusCode)
}
```

### 3. UDP 数据包发送

```go
// 创建UDP代理
udpProxy, err := client.UdpProxy("8.8.8.8", 53)
if err != nil {
    log.Printf("UDP代理创建失败: %v", err)
    return
}
defer udpProxy.Close()

// 发送DNS查询包
dnsQuery := []byte{/* DNS查询数据 */}
response, err := udpProxy.SendAndReceiveUdpPacket(dnsQuery)
if err != nil {
    log.Printf("UDP数据包发送失败: %v", err)
} else {
    fmt.Printf("收到响应: %x\n", response)
}
```

## 日志系统

### 日志格式

项目使用统一的日志格式，方便调试和监控：

- **服务器日志**: `[SOCKS5-SERVER]` - 服务器端操作日志
- **客户端日志**: `[SOCKS5-CLIENT]` - 客户端操作日志
- **工具日志**: `[SOCKS5-UTILS]` - 工具函数日志

### 日志内容

1. **连接管理**

   - 服务器启动/停止
   - 客户端连接/断开
   - 连接状态变化

2. **认证流程**

   - 认证方法协商
   - 用户名密码验证
   - 认证结果

3. **协议处理**

   - SOCKS5 请求/响应
   - 地址解析过程
   - 数据转发状态

4. **错误处理**
   - 连接失败
   - 协议错误
   - 认证失败

## 错误处理

项目实现了完善的错误处理机制：

1. **连接错误**

   - 网络连接失败
   - 目标服务器不可达
   - 连接超时

2. **协议错误**

   - 无效的 SOCKS5 请求
   - 不支持的地址类型
   - 认证失败

3. **资源管理**
   - 连接自动关闭
   - 内存泄漏防护
   - 并发安全

## 性能优化

1. **并发处理**

   - 每个客户端连接使用独立的 goroutine
   - 非阻塞 I/O 操作
   - 连接池管理

2. **内存管理**

   - 及时释放连接资源
   - 避免内存泄漏
   - 合理的缓冲区大小

3. **网络优化**
   - 高效的数据转发
   - 最小化网络延迟
   - 支持大文件传输

## 安全考虑

1. **认证机制**

   - 支持用户名密码认证
   - 防止未授权访问
   - 可扩展的认证方式

2. **访问控制**

   - 黑名单功能
   - IP 地址过滤
   - 连接限制

3. **数据安全**
   - 传输数据加密（可扩展）
   - 敏感信息保护
   - 日志安全

## 扩展性

项目设计具有良好的扩展性：

1. **协议扩展**

   - 支持更多 SOCKS5 命令
   - 可添加自定义协议
   - 插件化架构

2. **认证扩展**

   - 支持多种认证方式
   - 可集成外部认证系统
   - 动态用户管理

3. **功能扩展**
   - 负载均衡
   - 高可用性
   - 监控和统计

## 测试

### 运行测试

```bash
# 运行基本测试
go run main.go

# 预期输出
=== 启动SOCKS5服务器 ===
[SOCKS5-SERVER] 启动服务器 0.0.0.0:1080
[SOCKS5-SERVER] 服务器启动成功，等待连接...

=== 测试连接到本地服务器 ===
[SOCKS5-CLIENT] 开始认证，用户名: test, 密码: test
[SOCKS5-SERVER] 新连接: 127.0.0.1:60464
[SOCKS5-SERVER] 认证成功
[SOCKS5-CLIENT] 开始发送SOCKS5请求: cmd=1, host=www.baidu.com, port=80
[SOCKS5-UTILS] 解析地址: addrType=3, data=0e7777772e62616964752e636f6d0050
[SOCKS5-UTILS] 域名解析结果: www.baidu.com:80
[SOCKS5-SERVER] 处理CONNECT请求，目标: www.baidu.com:80
[SOCKS5-SERVER] 连接目标成功
[SOCKS5-SERVER] CONNECT响应已发送，开始转发数据
测试TCP连接成功
测试UDP连接成功
```

## 贡献指南

欢迎提交 Issue 和 Pull Request 来改进项目：

1. **代码规范**

   - 遵循 Go 语言规范
   - 添加适当的注释
   - 编写单元测试

2. **功能改进**

   - 新功能需求
   - 性能优化
   - 安全增强

3. **文档完善**
   - 更新 README
   - 添加使用示例
   - 完善 API 文档

## 许可证

本项目采用 MIT 许可证，详见 LICENSE 文件。

## 联系方式

如有问题或建议，请通过以下方式联系：

- 提交 GitHub Issue
- 发送邮件至项目维护者
- 参与项目讨论

---

**注意**: 本项目仅供学习和研究使用，请遵守相关法律法规。
