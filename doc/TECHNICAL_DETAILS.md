# SOCKS5 代理服务器技术实现细节

## 协议实现分析

### 1. SOCKS5 协议状态机

```
客户端连接
    ↓
认证协商阶段
    ↓
[无认证] → 直接进入请求阶段
[用户名密码认证] → 认证验证 → 请求阶段
    ↓
请求处理阶段
    ↓
[CONNECT] → 建立目标连接 → 数据转发
[BIND] → 等待连接 → 数据转发
[UDP] → UDP关联 → 数据转发
```

### 2. 认证阶段详细实现

#### 2.1 认证方法协商

**客户端发送：**

```go
// AuthPackage.toData() 生成
[0x05, 0x02, 0x00, 0x02]
// 版本号 | 方法数量 | 无认证 | 用户名密码认证
```

**服务器响应：**

```go
// 选择认证方法
if 支持无认证 {
    [0x05, 0x00]  // 选择无认证
} else if 支持用户名密码认证 {
    [0x05, 0x02]  // 选择用户名密码认证
} else {
    [0x05, 0xFF]  // 无支持的认证方法
}
```

#### 2.2 用户名密码认证

**客户端发送：**

```go
// 认证数据格式
[0x01, 0x04, 't', 'e', 's', 't', 0x04, 't', 'e', 's', 't']
// 子版本 | 用户名长度 | 用户名 | 密码长度 | 密码
```

**服务器验证：**

```go
// 验证用户名密码
if 用户名存在 && 密码匹配 {
    [0x01, 0x00]  // 认证成功
} else {
    [0x01, 0x01]  // 认证失败
}
```

### 3. 请求阶段详细实现

#### 3.1 请求格式解析

**客户端请求格式：**

```
+----+-----+-------+------+----------+----------+
|VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
+----+-----+-------+------+----------+----------+
| 1  |  1  | X'00' |  1   | Variable |    2     |
+----+-----+-------+------+----------+----------+
```

**代码实现：**

```go
func (s *Server) newConn(conn net.Conn) {
    // 读取请求头
    bytes := make([]byte, 1024)
    read, err := conn.Read(bytes)
    if err != nil {
        return
    }
    bytes = bytes[:read]

    // 验证最小长度
    if len(bytes) < 5 {
        return
    }

    // 解析命令和地址类型
    cmd := bytes[1]           // 命令类型
    addrType := bytes[3]      // 地址类型

    // 解析目标地址
    array, err := addressResolutionFormByteArray(bytes[4:], addrType)
    if err != nil {
        return
    }

    // 根据命令类型处理
    switch cmd {
    case Connect:
        s.handleConnect(conn, array)
    case Bind:
        s.handleBind(conn, array)
    case UDP:
        s.handleUDP(conn, array)
    }
}
```

#### 3.2 CONNECT 命令处理

**服务器处理流程：**

```go
func (s *Server) handleConnect(conn net.Conn, targetAddr string) {
    // 1. 建立到目标服务器的连接
    dial, err := net.Dial("tcp", targetAddr)
    if err != nil {
        // 发送失败响应
        conn.Write([]byte{Version, 0x05, Zero, IPv4, 0, 0, 0, 0, 0, 0})
        return
    }
    defer dial.Close()

    // 2. 获取本地绑定地址
    localAddr := dial.LocalAddr().String()
    split := strings.Split(localAddr, ":")
    port := split[len(split)-1]
    parseUint, _ := strconv.ParseUint(port, 10, 16)

    // 3. 解析本地地址类型
    Type, host, _ := addressResolution(strings.Join(split[:len(split)-1], ":"))

    // 4. 构造成功响应
    p := make([]byte, 2)
    binary.BigEndian.PutUint16(p, uint16(parseUint))
    data := []byte{Version, Zero, Zero, Type}
    data = append(data, host...)
    data = append(data, p...)

    // 5. 发送成功响应
    conn.Write(data)

    // 6. 开始数据转发
    ioCopy(conn, dial)
}
```

### 4. 地址解析实现

#### 4.1 地址类型支持

**IPv4 地址：**

```go
case IPv4:
    if len(data) < 4 {
        return "", fmt.Errorf("IPv4地址长度不足")
    }
    ip := net.IP(data[:4])
    if len(data) < 6 {
        return "", fmt.Errorf("IPv4地址端口长度不足")
    }
    port := binary.BigEndian.Uint16(data[4:6])
    return fmt.Sprintf("%s:%d", ip.String(), port), nil
```

**域名地址：**

```go
case Domain:
    if len(data) < 1 {
        return "", fmt.Errorf("域名长度不足")
    }
    domainLen := int(data[0])
    if len(data) < 1+domainLen+2 {
        return "", fmt.Errorf("域名地址长度不足")
    }
    domain := string(data[1 : 1+domainLen])
    port := binary.BigEndian.Uint16(data[1+domainLen : 1+domainLen+2])
    return fmt.Sprintf("%s:%d", domain, port), nil
```

**IPv6 地址：**

```go
case IPv6:
    if len(data) < 16 {
        return "", fmt.Errorf("IPv6地址长度不足")
    }
    ip := net.IP(data[:16])
    if len(data) < 18 {
        return "", fmt.Errorf("IPv6地址端口长度不足")
    }
    port := binary.BigEndian.Uint16(data[16:18])
    return fmt.Sprintf("%s:%d", ip.String(), port), nil
```

#### 4.2 地址格式化

**地址解析函数：**

```go
func addressResolution(addr string) (byte, []byte, uint16) {
    host, portStr, err := net.SplitHostPort(addr)
    if err != nil {
        return 0, nil, 0
    }

    port, err := strconv.ParseUint(portStr, 10, 16)
    if err != nil {
        return 0, nil, 0
    }

    // 尝试解析为IP地址
    if ip := net.ParseIP(host); ip != nil {
        if ip.To4() != nil {
            // IPv4
            return IPv4, ip.To4(), uint16(port)
        } else {
            // IPv6
            return IPv6, ip.To16(), uint16(port)
        }
    }

    // 域名
    return Domain, []byte(host), uint16(port)
}
```

### 5. 数据转发实现

#### 5.1 双向数据转发

```go
func ioCopy(dst net.Conn, src net.Conn) {
    defer func() {
        dst.Close()
        src.Close()
    }()

    // 双向复制数据
    go func() {
        io.Copy(dst, src)
    }()

    io.Copy(src, dst)
}
```

**实现原理：**

1. 使用 `defer` 确保连接关闭
2. 启动 goroutine 处理一个方向的数据转发
3. 主 goroutine 处理另一个方向的数据转发
4. 当任一方向的数据流结束时，两个连接都会被关闭

### 6. 客户端实现分析

#### 6.1 客户端连接建立

```go
func (c *Client) conn() (net.Conn, error) {
    // 1. 建立TCP连接
    conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port))
    if err != nil {
        return nil, err
    }

    // 2. 执行认证
    err = c.auth(conn)
    if err != nil {
        conn.Close()
        return nil, err
    }

    return conn, nil
}
```

#### 6.2 认证包封装

```go
type AuthPackage struct {
    methods []uint8
}

func (a *AuthPackage) addMethod(method uint8) {
    a.methods = append(a.methods, method)
}

func (a *AuthPackage) toData() []byte {
    data := []byte{Version, uint8(len(a.methods))}
    data = append(data, a.methods...)
    return data
}
```

#### 6.3 代理请求发送

```go
func (c *Client) requisition(conn net.Conn, host string, port uint16, cmd uint8) (net.Conn, error) {
    // 1. 解析目标地址
    Type, addr, _ := addressResolution(fmt.Sprintf("%s:%d", host, port))
    if Type == 0 {
        return nil, errors.New("addressResolution 失败")
    }

    // 2. 构造请求包
    buffer := bytes.Buffer{}
    buffer.Write([]byte{Version, cmd, Zero, Type})
    buffer.Write(addr)
    buffer.Write(portToBytes(port))

    // 3. 发送请求
    _, err := conn.Write(buffer.Bytes())
    if err != nil {
        return nil, err
    }

    // 4. 读取响应
    buf := make([]byte, 1024)
    read, err := conn.Read(buf)
    if err != nil {
        return nil, err
    }

    rdata := buf[:read]
    if len(rdata) < 2 || rdata[1] != Zero {
        return nil, errors.New(fmt.Sprintf("请求错误:%x", rdata[1]))
    }

    // 5. 处理UDP关联
    if cmd == UDP {
        return c.handleUDPAssociation(conn, rdata)
    }

    return nil, nil
}
```

### 7. 错误处理机制

#### 7.1 错误类型定义

```go
// 认证相关错误
var (
    ErrAuthDataLength = errors.New("认证数据长度不符")
    ErrProtocolMismatch = errors.New("协议不符合")
    ErrNoSupportedAuth = errors.New("没有支持的认证方法")
    ErrAuthFailed = errors.New("用户名或密码错误")
    ErrUnsupportedAuth = errors.New("不支持的认证方法")
)

// 地址解析错误
var (
    ErrIPv4Length = errors.New("IPv4地址长度不足")
    ErrIPv6Length = errors.New("IPv6地址长度不足")
    ErrDomainLength = errors.New("域名长度不足")
    ErrUnsupportedAddrType = errors.New("不支持的地址类型")
)
```

#### 7.2 错误处理策略

1. **连接级错误处理：**

   - 网络连接错误：记录日志并关闭连接
   - 协议错误：发送错误响应并关闭连接
   - 认证错误：发送认证失败响应

2. **应用级错误处理：**
   - 地址解析错误：返回错误信息
   - 目标连接失败：发送连接失败响应
   - 数据转发错误：关闭相关连接

### 8. 性能优化策略

#### 8.1 内存管理

```go
// 使用固定大小的缓冲区
bytes := make([]byte, 1024)
read, err := conn.Read(bytes)
bytes = bytes[:read]  // 截取实际读取的数据
```

#### 8.2 并发处理

```go
// 为每个客户端连接启动独立的goroutine
for s.listen != nil {
    accept, err := s.listen.Accept()
    if err == nil {
        go s.newConn(accept)  // 并发处理
    }
}
```

#### 8.3 连接管理

```go
// 使用defer确保连接关闭
defer func() {
    if conn != nil {
        conn.Close()
    }
}()
```

### 9. 安全考虑

#### 9.1 输入验证

```go
// 验证协议版本
if bytes[0] != Version {
    return errors.New("协议不符合")
}

// 验证数据长度
if len(bytes) < 3 {
    return errors.New("认证数据长度不符")
}
```

#### 9.2 认证机制

```go
// 用户名密码验证
if pas, has := s.UserMap[string(username)]; has {
    if pas == string(password) {
        conn.Write([]byte{0x01, Zero})
        return nil
    }
}
// 认证失败
conn.Write([]byte{0x01, 0x01})
return errors.New("用户名或密码错误")
```

### 10. 测试用例

#### 10.1 单元测试

```go
func TestAddressResolution(t *testing.T) {
    tests := []struct {
        input    string
        expected byte
        addr     []byte
        port     uint16
    }{
        {"127.0.0.1:8080", IPv4, []byte{127, 0, 0, 1}, 8080},
        {"example.com:80", Domain, []byte("example.com"), 80},
        {"::1:8080", IPv6, net.ParseIP("::1").To16(), 8080},
    }

    for _, test := range tests {
        addrType, addr, port := addressResolution(test.input)
        if addrType != test.expected {
            t.Errorf("地址类型不匹配: 期望 %d, 实际 %d", test.expected, addrType)
        }
        if port != test.port {
            t.Errorf("端口不匹配: 期望 %d, 实际 %d", test.port, port)
        }
    }
}
```

#### 10.2 集成测试

```go
func TestSOCKS5Server(t *testing.T) {
    // 启动测试服务器
    server := &Server{
        Config: Config{
            Host:     "127.0.0.1",
            Port:     0,  // 随机端口
            AuthList: []uint8{NoAuthenticationRequired},
        },
        UserMap: map[string]string{},
    }

    go server.Start()

    // 创建测试客户端
    client := &Client{
        Host: "127.0.0.1",
        Port: server.Config.Port,
    }

    // 测试连接
    conn, err := client.TcpProxy("example.com", 80)
    if err != nil {
        t.Errorf("连接失败: %v", err)
    }
    defer conn.Close()
}
```

### 11. 部署和运维

#### 11.1 配置文件

```yaml
# config.yaml
server:
  host: "0.0.0.0"
  port: 1088
  auth_methods: [0, 2] # 无认证, 用户名密码认证

users:
  test: "test"
  admin: "admin123"

blacklist:
  - "192.168.1.100"
  - "malicious.com"

logging:
  level: "info"
  file: "socks5.log"
```

#### 11.2 监控指标

```go
type Metrics struct {
    ActiveConnections int64
    TotalConnections  int64
    BytesTransferred  int64
    Errors           int64
}

func (s *Server) collectMetrics() {
    // 收集连接数
    atomic.AddInt64(&s.metrics.ActiveConnections, 1)
    defer atomic.AddInt64(&s.metrics.ActiveConnections, -1)

    // 收集传输字节数
    // 收集错误数
}
```

### 12. 扩展功能

#### 12.1 插件系统

```go
type Plugin interface {
    OnConnect(conn net.Conn) error
    OnRequest(req *Request) error
    OnResponse(resp *Response) error
}

type Request struct {
    Command byte
    Address string
    Port    uint16
}

type Response struct {
    Status byte
    Address string
    Port    uint16
}
```

#### 12.2 负载均衡

```go
type LoadBalancer struct {
    servers []*Server
    current int
    mu      sync.Mutex
}

func (lb *LoadBalancer) GetNext() *Server {
    lb.mu.Lock()
    defer lb.mu.Unlock()

    server := lb.servers[lb.current]
    lb.current = (lb.current + 1) % len(lb.servers)
    return server
}
```

## 总结

本 SOCKS5 代理服务器实现遵循 RFC 1928 和 RFC 1929 标准，提供了完整的协议支持。通过模块化设计和良好的错误处理，确保了系统的稳定性和可维护性。

主要技术特点：

- 完整的 SOCKS5 协议实现
- 高效的并发处理
- 完善的错误处理
- 良好的扩展性
- 详细的文档和测试

该实现可以作为学习网络编程和代理技术的优秀示例，也可以作为生产环境的基础框架。
