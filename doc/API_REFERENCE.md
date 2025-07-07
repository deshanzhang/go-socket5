# SOCKS5 代理服务器 API 参考文档

## 概述

本文档详细说明了 SOCKS5 代理服务器的所有公开 API 接口，包括服务器端和客户端的完整 API 参考。

## 服务器端 API

### Server 结构体

```go
type Server struct {
    listen  net.Listener    // TCP监听器
    Config  Config          // 服务器配置
    UserMap map[string]string // 用户认证映射
}
```

#### 字段说明

- `listen`: 内部 TCP 监听器，用于接受客户端连接
- `Config`: 服务器配置信息
- `UserMap`: 用户名到密码的映射，用于认证

### Config 结构体

```go
type Config struct {
    Host      string   // 监听地址
    Port      uint16   // 监听端口
    BlackList []string // 黑名单
    AuthList  []uint8  // 支持的认证方法
}
```

#### 字段说明

- `Host`: 服务器监听地址，如 "0.0.0.0" 或 "127.0.0.1"
- `Port`: 服务器监听端口，如 1088
- `BlackList`: IP 地址或域名黑名单
- `AuthList`: 支持的认证方法列表

### 服务器方法

#### Start() error

启动 SOCKS5 服务器。

**返回值：**

- `error`: 启动失败时返回错误信息

**示例：**

```go
server := &Server{
    Config: Config{
        Host:     "0.0.0.0",
        Port:     1088,
        AuthList: []uint8{NoAuthenticationRequired},
    },
    UserMap: map[string]string{},
}

err := server.Start()
if err != nil {
    log.Fatal(err)
}
```

#### newConn(conn net.Conn)

处理客户端连接（内部方法）。

**参数：**

- `conn net.Conn`: 客户端连接

**示例：**

```go
// 此方法由Start()自动调用，通常不需要手动调用
```

#### auth(conn net.Conn) error

执行客户端认证（内部方法）。

**参数：**

- `conn net.Conn`: 客户端连接

**返回值：**

- `error`: 认证失败时返回错误信息

**认证流程：**

1. 接收客户端支持的认证方法
2. 选择服务器支持的认证方法
3. 如果选择用户名密码认证，验证用户凭据

## 客户端 API

### Client 结构体

```go
type Client struct {
    Host     string // 代理服务器地址
    Port     uint16 // 代理服务器端口
    UserName string // 用户名
    Password string // 密码
}
```

#### 字段说明

- `Host`: SOCKS5 代理服务器地址
- `Port`: SOCKS5 代理服务器端口
- `UserName`: 认证用户名（可选）
- `Password`: 认证密码（可选）

### AuthPackage 结构体

```go
type AuthPackage struct {
    methods []uint8 // 认证方法列表
}
```

#### 方法

##### addMethod(method uint8)

添加认证方法。

**参数：**

- `method uint8`: 认证方法，如 `NoAuthenticationRequired` 或 `AccountPasswordAuthentication`

**示例：**

```go
auth := &AuthPackage{}
auth.addMethod(NoAuthenticationRequired)
auth.addMethod(AccountPasswordAuthentication)
```

##### toData() []byte

将认证包转换为字节数组。

**返回值：**

- `[]byte`: SOCKS5 认证方法协商数据

**示例：**

```go
data := auth.toData()
conn.Write(data)
```

### UdpProxy 结构体

```go
type UdpProxy struct {
    Conn net.Conn // UDP连接
    Host string   // 目标主机
    Port uint16   // 目标端口
}
```

#### 字段说明

- `Conn`: 到 SOCKS5 服务器的 UDP 连接
- `Host`: 目标服务器地址
- `Port`: 目标服务器端口

### 客户端方法

#### conn() (net.Conn, error)

建立到 SOCKS5 服务器的连接。

**返回值：**

- `net.Conn`: 到代理服务器的连接
- `error`: 连接失败时返回错误信息

**示例：**

```go
conn, err := client.conn()
if err != nil {
    return err
}
defer conn.Close()
```

#### auth(conn net.Conn) error

执行认证流程（内部方法）。

**参数：**

- `conn net.Conn`: 到代理服务器的连接

**返回值：**

- `error`: 认证失败时返回错误信息

**认证流程：**

1. 发送支持的认证方法
2. 接收服务器选择的认证方法
3. 如果选择用户名密码认证，发送用户凭据

#### requisition(conn net.Conn, host string, port uint16, cmd uint8) (net.Conn, error)

发送代理请求（内部方法）。

**参数：**

- `conn net.Conn`: 到代理服务器的连接
- `host string`: 目标主机地址
- `port uint16`: 目标端口
- `cmd uint8`: 命令类型（Connect, Bind, UDP）

**返回值：**

- `net.Conn`: 对于 UDP 命令，返回 UDP 连接
- `error`: 请求失败时返回错误信息

#### TcpProxy(host string, port uint16) (net.Conn, error)

建立 TCP 代理连接。

**参数：**

- `host string`: 目标主机地址
- `port uint16`: 目标端口

**返回值：**

- `net.Conn`: 到目标服务器的连接
- `error`: 连接失败时返回错误信息

**示例：**

```go
conn, err := client.TcpProxy("example.com", 80)
if err != nil {
    log.Printf("TCP代理失败: %v", err)
    return
}
defer conn.Close()

// 使用连接发送数据
conn.Write([]byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"))
```

#### UdpProxy(host string, port uint16) (*UdpProxy, error)

建立 UDP 代理连接。

**参数：**

- `host string`: 目标主机地址
- `port uint16`: 目标端口

**返回值：**

- `*UdpProxy`: UDP 代理对象
- `error`: 连接失败时返回错误信息

**示例：**

```go
udpProxy, err := client.UdpProxy("8.8.8.8", 53)
if err != nil {
    log.Printf("UDP代理失败: %v", err)
    return
}
defer udpProxy.Conn.Close()

// 发送DNS查询
query := []byte{/* DNS查询数据 */}
udpProxy.Conn.Write(query)
```

#### GetHttpProxyClient() *http.Client

获取配置了 SOCKS5 代理的 HTTP 客户端。

**返回值：**

- `*http.Client`: 配置了代理的 HTTP 客户端

**示例：**

```go
httpClient := client.GetHttpProxyClient()

resp, err := httpClient.Get("http://httpbin.org/ip")
if err != nil {
    log.Printf("HTTP请求失败: %v", err)
    return
}
defer resp.Body.Close()

body, _ := io.ReadAll(resp.Body)
fmt.Printf("响应: %s\n", string(body))
```

## 工具函数 API

### addressResolution(addr string) (byte, []byte, uint16)

解析地址字符串。

**参数：**

- `addr string`: 地址字符串，格式为 "host:port"

**返回值：**

- `byte`: 地址类型（IPv4, IPv6, Domain）
- `[]byte`: 地址字节数据
- `uint16`: 端口号

**示例：**

```go
addrType, addrBytes, port := addressResolution("example.com:80")
fmt.Printf("地址类型: %d, 地址: %v, 端口: %d\n", addrType, addrBytes, port)
```

### addressResolutionFormByteArray(data []byte, addrType byte) (string, error)

从字节数组解析地址。

**参数：**

- `data []byte`: 地址字节数据
- `addrType byte`: 地址类型

**返回值：**

- `string`: 格式化的地址字符串
- `error`: 解析失败时返回错误信息

**示例：**

```go
addr, err := addressResolutionFormByteArray(data, IPv4)
if err != nil {
    log.Printf("地址解析失败: %v", err)
    return
}
fmt.Printf("解析结果: %s\n", addr)
```

### ioCopy(dst net.Conn, src net.Conn)

在两个连接之间复制数据。

**参数：**

- `dst net.Conn`: 目标连接
- `src net.Conn`: 源连接

**示例：**

```go
// 双向数据转发
go ioCopy(clientConn, targetConn)
ioCopy(targetConn, clientConn)
```

## 常量定义

### 协议常量

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

### 日志前缀

```go
const (
    LogPrefixServer = "[SOCKS5-SERVER]"  // 服务器日志前缀
    LogPrefixClient = "[SOCKS5-CLIENT]"  // 客户端日志前缀
    LogPrefixUtils  = "[SOCKS5-UTILS]"   // 工具函数日志前缀
)
```

## 错误处理

### 常见错误类型

```go
var (
    ErrAuthFailed = errors.New("认证失败")
    ErrConnectionFailed = errors.New("连接失败")
    ErrInvalidAddress = errors.New("无效地址")
    ErrUnsupportedCommand = errors.New("不支持的命令")
)
```

### 错误处理示例

```go
conn, err := client.TcpProxy("example.com", 80)
if err != nil {
    switch {
    case errors.Is(err, ErrAuthFailed):
        log.Printf("认证失败，请检查用户名和密码")
    case errors.Is(err, ErrConnectionFailed):
        log.Printf("连接失败，请检查网络和服务器状态")
    case errors.Is(err, ErrInvalidAddress):
        log.Printf("无效地址，请检查目标地址格式")
    default:
        log.Printf("未知错误: %v", err)
    }
    return
}
```

## 使用示例

### 基本服务器启动

```go
package main

import (
    "log"
    "go-socket5/socket5"
)

func main() {
    server := &socks5.Server{
        Config: socks5.Config{
            Host:     "0.0.0.0",
            Port:     1088,
            AuthList: []uint8{socks5.AccountPasswordAuthentication},
        },
        UserMap: map[string]string{
            "test": "test",
        },
    }

    log.Printf("启动SOCKS5服务器 %s:%d", server.Config.Host, server.Config.Port)
    err := server.Start()
    if err != nil {
        log.Fatal(err)
    }
}
```

### 基本客户端使用

```go
package main

import (
    "fmt"
    "log"
    "go-socket5/socket5"
)

func main() {
    client := &socks5.Client{
        Host:     "127.0.0.1",
        Port:     1088,
        UserName: "test",
        Password: "test",
    }

    // TCP代理
    conn, err := client.TcpProxy("example.com", 80)
    if err != nil {
        log.Printf("TCP代理失败: %v", err)
        return
    }
    defer conn.Close()

    fmt.Println("TCP代理连接成功")

    // HTTP代理
    httpClient := client.GetHttpProxyClient()
    resp, err := httpClient.Get("http://httpbin.org/ip")
    if err != nil {
        log.Printf("HTTP请求失败: %v", err)
        return
    }
    defer resp.Body.Close()

    fmt.Printf("HTTP状态码: %d\n", resp.StatusCode)
}
```

### UDP代理使用

```go
package main

import (
    "fmt"
    "log"
    "go-socket5/socket5"
)

func main() {
    client := &socks5.Client{
        Host:     "127.0.0.1",
        Port:     1088,
        UserName: "test",
        Password: "test",
    }

    // UDP代理
    udpProxy, err := client.UdpProxy("8.8.8.8", 53)
    if err != nil {
        log.Printf("UDP代理失败: %v", err)
        return
    }
    defer udpProxy.Conn.Close()

    // 发送DNS查询
    dnsQuery := []byte{/* DNS查询数据 */}
    _, err = udpProxy.Conn.Write(dnsQuery)
    if err != nil {
        log.Printf("发送DNS查询失败: %v", err)
        return
    }

    // 读取响应
    buffer := make([]byte, 512)
    n, err := udpProxy.Conn.Read(buffer)
    if err != nil {
        log.Printf("读取DNS响应失败: %v", err)
        return
    }

    fmt.Printf("收到DNS响应，长度: %d\n", n)
}
```

## 最佳实践

### 1. 连接管理

```go
// 总是使用defer关闭连接
conn, err := client.TcpProxy("example.com", 80)
if err != nil {
    return err
}
defer conn.Close()

// 设置连接超时
conn.SetDeadline(time.Now().Add(30 * time.Second))
```

### 2. 错误处理

```go
// 检查具体错误类型
if err != nil {
    if errors.Is(err, ErrAuthFailed) {
        // 处理认证错误
    } else if errors.Is(err, ErrConnectionFailed) {
        // 处理连接错误
    } else {
        // 处理其他错误
    }
}
```

### 3. 资源清理

```go
// 使用defer确保资源清理
func handleConnection(conn net.Conn) {
    defer conn.Close()
    defer log.Printf("连接已关闭: %s", conn.RemoteAddr())
    
    // 处理连接
}
```

### 4. 日志记录

```go
// 使用统一的日志前缀
log.Printf("%s 新连接: %s", LogPrefixServer, conn.RemoteAddr())
log.Printf("%s 认证成功", LogPrefixServer)
log.Printf("%s 连接目标: %s", LogPrefixServer, targetAddr)
```

## 性能优化

### 1. 连接池

```go
// 使用连接池减少连接开销
var connPool = sync.Pool{
    New: func() interface{} {
        return &socks5.Client{}
    },
}

func getClient() *socks5.Client {
    return connPool.Get().(*socks5.Client)
}

func putClient(client *socks5.Client) {
    connPool.Put(client)
}
```

### 2. 缓冲区优化

```go
// 使用固定大小的缓冲区
const bufferSize = 4096
buffer := make([]byte, bufferSize)

for {
    n, err := conn.Read(buffer)
    if err != nil {
        break
    }
    
    _, err = targetConn.Write(buffer[:n])
    if err != nil {
        break
    }
}
```

### 3. 并发控制

```go
// 限制并发连接数
var semaphore = make(chan struct{}, 100)

func handleRequest() {
    semaphore <- struct{}{} // 获取信号量
    defer func() {
        <-semaphore // 释放信号量
    }()
    
    // 处理请求
}
```

---

**注意：** 本文档提供了完整的API参考，适合开发者集成和使用SOCKS5代理服务器。
