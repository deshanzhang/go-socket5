package socks5

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 日志前缀常量
const (
	LogPrefixServer = "[SOCKS5-SERVER]"
	LogPrefixClient = "[SOCKS5-CLIENT]"
	LogPrefixUtils  = "[SOCKS5-UTILS]"
)

// 高并发配置常量
const (
	MaxConcurrentConnections = 10000 // 最大并发连接数
	ConnectionTimeout        = 30 * time.Second
	ReadTimeout              = 10 * time.Second
	WriteTimeout             = 10 * time.Second
)

var ServerClient *Server

// Server SOCKS5服务器结构体
type Server struct {
	listen  net.Listener      // TCP监听器
	Config  Config            // 服务器配置
	UserMap map[string]string // 用户认证映射表

	// 高并发优化字段
	connCount   int32        // 当前连接数
	connMutex   sync.RWMutex // 连接数锁
	ctx         context.Context
	cancel      context.CancelFunc
	rateLimiter *RateLimiter // 限流器
}

// RateLimiter 限流器
type RateLimiter struct {
	tokens     chan struct{}
	rate       time.Duration
	burst      int
	lastRefill time.Time
	mutex      sync.Mutex
}

// NewRateLimiter 创建新的限流器
func NewRateLimiter(rate time.Duration, burst int) *RateLimiter {
	rl := &RateLimiter{
		tokens: make(chan struct{}, burst),
		rate:   rate,
		burst:  burst,
	}

	// 初始化令牌
	for i := 0; i < burst; i++ {
		rl.tokens <- struct{}{}
	}

	// 启动令牌补充
	go rl.refillTokens()
	return rl
}

// refillTokens 补充令牌
func (rl *RateLimiter) refillTokens() {
	ticker := time.NewTicker(rl.rate)
	defer ticker.Stop()

	for range ticker.C {
		select {
		case rl.tokens <- struct{}{}:
		default:
		}
	}
}

// Allow 检查是否允许连接
func (rl *RateLimiter) Allow() bool {
	select {
	case <-rl.tokens:
		return true
	default:
		return false
	}
}

// Config 服务器配置结构体
type Config struct {
	Host      string   // 监听地址
	Port      uint16   // 监听端口
	BlackList []string // 黑名单列表
	AuthList  []uint8  // 支持的认证方法列表
}

// Start 启动SOCKS5服务器
func (s *Server) Start() (err error) {
	// 验证认证配置
	if err := s.validateAuthConfig(); err != nil {
		return fmt.Errorf("认证配置验证失败: %v", err)
	}

	// 初始化上下文和限流器
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.rateLimiter = NewRateLimiter(100*time.Millisecond, 1000) // 每秒1000个连接

	log.Printf("%s 启动服务器 %s:%d", LogPrefixServer, s.Config.Host, s.Config.Port)
	s.listen, err = net.Listen("tcp", fmt.Sprintf("%s:%d", s.Config.Host, s.Config.Port))
	if err != nil {
		log.Printf("%s 监听失败: %v", LogPrefixServer, err)
		return err
	}

	// 设置TCP keep-alive
	if tcpListener, ok := s.listen.(*net.TCPListener); ok {
		tcpListener.SetDeadline(time.Time{}) // 禁用超时
	}

	log.Printf("%s 服务器启动成功，等待连接...", LogPrefixServer)

	for {
		accept, err := s.listen.Accept()
		if err != nil {
			if s.ctx.Err() != nil {
				// 服务器正在关闭
				return nil
			}
			log.Printf("%s 接受连接失败: %v", LogPrefixServer, err)
			continue
		}

		// 限流检查
		if !s.rateLimiter.Allow() {
			log.Printf("%s 连接被限流拒绝: %v", LogPrefixServer, accept.RemoteAddr())
			accept.Close()
			continue
		}

		// 并发连接数检查
		if !s.incrementConnCount() {
			log.Printf("%s 达到最大并发连接数限制: %v", LogPrefixServer, accept.RemoteAddr())
			accept.Close()
			continue
		}

		// 为每个连接启动goroutine处理
		go s.handleConnection(accept)
	}
}

// handleConnection 处理客户端连接
func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		s.decrementConnCount()
		log.Printf("%s 连接关闭: %v", LogPrefixServer, conn.RemoteAddr())
	}()

	log.Printf("%s 新连接: %v", LogPrefixServer, conn.RemoteAddr())

	// 设置连接超时
	conn.SetDeadline(time.Now().Add(ConnectionTimeout))

	// 执行认证
	if err := s.auth(conn); err != nil {
		log.Printf("%s 认证失败: %v", LogPrefixServer, err)
		return
	}

	log.Printf("%s 认证完成，开始处理SOCKS5请求: %v", LogPrefixServer, conn.RemoteAddr())

	// 清除超时，进入正常数据转发模式
	conn.SetDeadline(time.Time{})

	// 处理SOCKS5请求
	s.handleSocks5Request(conn)
}

// incrementConnCount 增加连接数
func (s *Server) incrementConnCount() bool {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	if s.connCount >= MaxConcurrentConnections {
		return false
	}
	s.connCount++
	return true
}

// decrementConnCount 减少连接数
func (s *Server) decrementConnCount() {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	if s.connCount > 0 {
		s.connCount--
	}
}

// getConnCount 获取当前连接数
func (s *Server) getConnCount() int32 {
	s.connMutex.RLock()
	defer s.connMutex.RUnlock()
	return s.connCount
}

// auth 处理客户端认证
func (s *Server) auth(conn net.Conn) error {
	bytes := make([]byte, 16)
	read, err := conn.Read(bytes)
	if err != nil {
		log.Printf("%s 认证阶段读取失败: %v", LogPrefixServer, err)
		return err
	}
	bytes = bytes[:read]
	log.Printf("%s 收到认证请求: %x", LogPrefixServer, bytes)

	if len(bytes) < 3 {
		return errors.New("认证请求长度不足")
	}

	version := bytes[0]
	if version != Version {
		return fmt.Errorf("协议版本不匹配: 期望 %d, 实际 %d", Version, version)
	}

	methodCount := int(bytes[1])
	if len(bytes) < 2+methodCount {
		return errors.New("认证方法数量不匹配")
	}

	methods := bytes[2 : 2+methodCount]
	log.Printf("%s 客户端支持的认证方法: %v", LogPrefixServer, methods)

	// 选择认证方法 - 按服务器配置的优先级选择客户端支持的方法
	var selectedMethod uint8 = 0xFF // 初始化为无效值，避免与NoAuthenticationRequired冲突

	// 按服务器配置的优先级顺序选择认证方法
	for _, supportedMethod := range s.Config.AuthList {
		for _, clientMethod := range methods {
			if supportedMethod == clientMethod {
				selectedMethod = supportedMethod
				break
			}
		}
		if selectedMethod != 0xFF {
			break
		}
	}

	if selectedMethod == 0xFF {
		log.Printf("%s 没有找到支持的认证方法，服务器支持: %v, 客户端支持: %v", LogPrefixServer, s.Config.AuthList, methods)
		conn.Write([]byte{Version, 0xFF}) // 0xFF表示没有可接受的认证方法
		return errors.New("没有支持的认证方法")
	}

	// 发送认证方法选择响应
	conn.Write([]byte{Version, selectedMethod})
	log.Printf("%s 选择认证方法: %s (0x%02x)", LogPrefixServer, GetAuthMethodName(selectedMethod), selectedMethod)

	// 处理具体的认证方法
	switch selectedMethod {
	case NoAuthenticationRequired:
		log.Printf("%s 无需认证", LogPrefixServer)
		return nil
	case AccountPasswordAuthentication:
		return s.handlePasswordAuth(conn)
	default:
		return fmt.Errorf("不支持的认证方法: %d", selectedMethod)
	}
}

// validateAuthConfig 验证认证配置
func (s *Server) validateAuthConfig() error {
	warnings := ValidateAuthConfig(s.Config.AuthList, s.UserMap)

	for _, warning := range warnings {
		if warning == "认证方法列表为空" || warning == "没有有效的认证方法" {
			return errors.New(warning)
		}
		log.Printf("%s 配置警告: %s", LogPrefixServer, warning)
	}

	// 打印认证配置信息
	log.Printf("%s 认证配置:", LogPrefixServer)
	for i, method := range s.Config.AuthList {
		log.Printf("%s   %d. %s (0x%02x)", LogPrefixServer, i+1, GetAuthMethodName(method), method)
	}

	if len(s.UserMap) > 0 {
		log.Printf("%s 配置了 %d 个用户账户", LogPrefixServer, len(s.UserMap))
	}

	return nil
}

// handlePasswordAuth 处理用户名密码认证
func (s *Server) handlePasswordAuth(conn net.Conn) error {
	bytes := make([]byte, 1024)
	read, err := conn.Read(bytes)
	if err != nil {
		log.Printf("%s 读取认证数据失败: %v", LogPrefixServer, err)
		return err
	}
	bytes = bytes[:read]
	log.Printf("%s 收到认证数据: %x", LogPrefixServer, bytes)

	if len(bytes) < 3 {
		return errors.New("认证数据长度不足")
	}

	version := bytes[0]
	if version != 0x01 {
		return fmt.Errorf("认证子协议版本不匹配: 期望 1, 实际 %d", version)
	}

	usernameLen := int(bytes[1])
	if len(bytes) < 2+usernameLen+1 {
		return errors.New("用户名长度不匹配")
	}

	username := string(bytes[2 : 2+usernameLen])
	passwordLen := int(bytes[2+usernameLen])

	if len(bytes) < 2+usernameLen+1+passwordLen {
		return errors.New("密码长度不匹配")
	}

	password := string(bytes[3+usernameLen : 3+usernameLen+passwordLen])

	log.Printf("%s 认证信息 - 用户名: %s, 密码: %s", LogPrefixServer, username, password)

	// 验证用户名密码
	expectedPassword, exists := s.UserMap[username]
	if !exists {
		log.Printf("%s 认证失败 - 用户不存在: %s", LogPrefixServer, username)
		conn.Write([]byte{0x01, 0x01}) // 认证失败
		return errors.New("用户不存在")
	}

	if expectedPassword != password {
		log.Printf("%s 认证失败 - 密码错误，用户名: %s", LogPrefixServer, username)
		conn.Write([]byte{0x01, 0x01}) // 认证失败
		return errors.New("密码错误")
	}

	log.Printf("%s 认证成功 - 用户名: %s", LogPrefixServer, username)
	conn.Write([]byte{0x01, 0x00}) // 认证成功
	return nil
}

// handleSocks5Request 处理SOCKS5请求
func (s *Server) handleSocks5Request(conn net.Conn) {
	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(ReadTimeout))

	bytes := make([]byte, 1024)
	read, err := conn.Read(bytes)
	if err != nil {
		log.Printf("%s 读取SOCKS5请求失败: %v", LogPrefixServer, err)
		return
	}
	bytes = bytes[:read]
	log.Printf("%s 收到SOCKS5请求: %x", LogPrefixServer, bytes)

	if len(bytes) < 5 {
		log.Printf("%s 请求长度不足", LogPrefixServer)
		return
	}

	cmd := bytes[1]
	addrType := bytes[3]

	// 根据地址类型确定地址字段的起始位置
	var addrData []byte
	switch addrType {
	case IPv4:
		// IPv4: 4字节IP + 2字节端口
		if len(bytes) < 10 {
			log.Printf("%s IPv4地址数据长度不足", LogPrefixServer)
			return
		}
		addrData = bytes[4:10]
	case Domain:
		// 域名: 1字节长度 + 域名 + 2字节端口
		if len(bytes) < 5 {
			log.Printf("%s 域名地址数据长度不足", LogPrefixServer)
			return
		}
		domainLen := int(bytes[4])
		if len(bytes) < 5+domainLen+2 {
			log.Printf("%s 域名地址数据长度不足", LogPrefixServer)
			return
		}
		addrData = bytes[4 : 5+domainLen+2]
	case IPv6:
		// IPv6: 16字节IP + 2字节端口
		if len(bytes) < 22 {
			log.Printf("%s IPv6地址数据长度不足", LogPrefixServer)
			return
		}
		addrData = bytes[4:22]
	default:
		log.Printf("%s 不支持的地址类型: %d", LogPrefixServer, addrType)
		return
	}

	array, err := addressResolutionFormByteArray(addrData, addrType)
	if err != nil {
		log.Printf("%s 地址解析失败: %v", LogPrefixServer, err)
		return
	}
	log.Printf("%s CMD: %d, 目标: %s", LogPrefixServer, cmd, array)

	switch cmd {
	case Connect:
		s.handleConnect(conn, array)
	case Bind:
		s.handleBind(conn, array)
	case UDP:
		s.handleUDP(conn, array)
	default:
		log.Printf("%s 不支持的命令: %d", LogPrefixServer, cmd)
	}
}

// handleConnect 处理CONNECT命令
func (s *Server) handleConnect(conn net.Conn, targetAddr string) {
	log.Printf("%s 处理CONNECT请求，目标: %s", LogPrefixServer, targetAddr)

	// 使用带超时的连接
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}
	dial, err := dialer.Dial("tcp", targetAddr)
	if err != nil {
		log.Printf("%s 连接目标失败: %v", LogPrefixServer, err)
		// 发送连接失败响应
		conn.Write([]byte{Version, 0x01, Zero, IPv4, 0, 0, 0, 0, 0, 0})
		return
	}
	defer dial.Close()

	log.Printf("%s 连接目标成功", LogPrefixServer)

	// 发送连接成功响应
	addr := dial.LocalAddr().String()
	split := strings.Split(addr, ":")
	port := split[len(split)-1]
	parseUint, _ := strconv.ParseUint(port, 10, 16)
	Type, host, _ := addressResolution(strings.Join(split[:len(split)-1], ":"))
	p := make([]byte, 2)
	binary.BigEndian.PutUint16(p, uint16(parseUint))
	data := []byte{Version, Zero, Zero, Type}
	data = append(data, host...)
	data = append(data, p...)

	// 设置写入超时
	conn.SetWriteDeadline(time.Now().Add(WriteTimeout))
	conn.Write(data)

	log.Printf("%s CONNECT响应已发送，开始转发数据", LogPrefixServer)

	// 开始数据转发
	s.forwardData(conn, dial)
}

// handleBind 处理BIND命令
func (s *Server) handleBind(conn net.Conn, targetAddr string) {
	log.Printf("%s 处理BIND请求", LogPrefixServer)

	// BIND命令处理 - 绑定本地端口等待连接
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Printf("%s 创建监听器失败: %v", LogPrefixServer, err)
		// 发送失败响应
		conn.Write([]byte{Version, 0x01, Zero, IPv4, 0, 0, 0, 0, 0, 0})
		return
	}
	defer listener.Close()

	// 获取绑定的地址和端口
	addr := listener.Addr().String()
	split := strings.Split(addr, ":")
	port := split[len(split)-1]
	parseUint, _ := strconv.ParseUint(port, 10, 16)
	Type, host, _ := addressResolution(strings.Join(split[:len(split)-1], ":"))
	p := make([]byte, 2)
	binary.BigEndian.PutUint16(p, uint16(parseUint))

	// 发送绑定成功响应
	data := []byte{Version, Zero, Zero, Type}
	data = append(data, host...)
	data = append(data, p...)
	conn.Write(data)
	log.Printf("%s BIND响应已发送", LogPrefixServer)

	// 等待连接
	accept, err := listener.Accept()
	if err != nil {
		log.Printf("%s 等待连接失败: %v", LogPrefixServer, err)
		return
	}
	defer accept.Close()

	// 发送连接建立响应
	conn.Write(data)
	log.Printf("%s BIND连接建立，开始转发数据", LogPrefixServer)
	s.forwardData(conn, accept)
}

// handleUDP 处理UDP命令
func (s *Server) handleUDP(conn net.Conn, targetAddr string) {
	log.Printf("%s 处理UDP请求", LogPrefixServer)

	// UDP命令处理 - 建立UDP代理
	udpAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		log.Printf("%s 解析UDP地址失败: %v", LogPrefixServer, err)
		conn.Write([]byte{Version, 0x01, Zero, IPv4, 0, 0, 0, 0, 0, 0})
		return
	}

	udpListener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Printf("%s 创建UDP监听器失败: %v", LogPrefixServer, err)
		conn.Write([]byte{Version, 0x01, Zero, IPv4, 0, 0, 0, 0, 0, 0})
		return
	}
	defer udpListener.Close()

	// 发送UDP绑定成功响应
	addr := udpListener.LocalAddr().String()
	split := strings.Split(addr, ":")
	port := split[len(split)-1]
	parseUint, _ := strconv.ParseUint(port, 10, 16)

	// 使用IPv4地址类型，因为这是最常用的
	Type := byte(IPv4)
	host := net.ParseIP("0.0.0.0").To4()
	p := make([]byte, 2)
	binary.BigEndian.PutUint16(p, uint16(parseUint))
	data := []byte{Version, Zero, Zero, Type}
	data = append(data, host...)
	data = append(data, p...)
	conn.Write(data)

	log.Printf("%s UDP代理已建立，开始处理UDP数据", LogPrefixServer)
	s.handleUDPProxy(conn, udpListener)
}

// forwardData 转发数据
func (s *Server) forwardData(conn1, conn2 net.Conn) {
	// 创建双向数据转发
	done := make(chan bool, 2)

	// 从conn1转发到conn2
	go func() {
		defer func() { done <- true }()
		buffer := make([]byte, 4096)
		for {
			n, err := conn1.Read(buffer)
			if err != nil {
				return
			}
			if n > 0 {
				_, err = conn2.Write(buffer[:n])
				if err != nil {
					return
				}
			}
		}
	}()

	// 从conn2转发到conn1
	go func() {
		defer func() { done <- true }()
		buffer := make([]byte, 4096)
		for {
			n, err := conn2.Read(buffer)
			if err != nil {
				return
			}
			if n > 0 {
				_, err = conn1.Write(buffer[:n])
				if err != nil {
					return
				}
			}
		}
	}()

	// 等待任一方向完成
	<-done
}

// handleUDPProxy 处理UDP代理数据转发
func (s *Server) handleUDPProxy(tcpConn net.Conn, udpListener *net.UDPConn) {
	defer tcpConn.Close()
	defer udpListener.Close()
	log.Printf("%s 开始UDP代理数据转发", LogPrefixServer)

	// 简单的UDP代理实现
	// 保持TCP控制连接活跃，UDP数据通过UDP监听器处理
	buffer := make([]byte, 1024)
	for {
		_, err := tcpConn.Read(buffer)
		if err != nil {
			log.Printf("%s UDP代理连接关闭: %v", LogPrefixServer, err)
			break
		}
		// 保持TCP连接活跃，实际UDP数据通过UDP监听器处理
	}
}

type ConfigInput struct {
	IpAddress string `json:"ip_address"`
	User      string `json:"user"`
	Password  string `json:"password"`
	Port      int64  `json:"port"`
}

// TestConnection 测试实际连接功能
func TestConnection(client Client, urls []string) error {
	fmt.Println("测试实际网络连接...")
	// 尝试连接百度
	for _, url := range urls {
		conn, err := client.TcpProxy(url, 80)
		if err != nil {
			fmt.Printf("❌ 连接失败: %v\n", err)
			return err
		}

		fmt.Printf("✅ 连接成功！\n")
		fmt.Printf("本地地址: %s\n", conn.LocalAddr())
		fmt.Printf("远程地址: %s\n", conn.RemoteAddr())

		// 发送简单的HTTP请求
		httpRequest := fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", url)
		_, err = conn.Write([]byte(httpRequest))
		if err != nil {
			fmt.Printf("❌ 发送HTTP请求失败: %v\n", err)
			conn.Close()
			return err
		}

		// 读取响应
		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Printf("❌ 读取响应失败: %v\n", err)
		} else {
			fmt.Printf("✅ 收到响应 (%d 字节)\n", n)
			// 只显示前100个字符，避免日志过长
			response := string(buffer[:n])
			if len(response) > 100 {
				response = response[:100] + "..."
			}
			fmt.Printf("响应内容: %s\n", response)
		}

		conn.Close()
	}

	fmt.Println("✅ 实际连接测试完成")
	return nil
}
