package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	socks5 "go-socket5/socket5"
	"io"
	"net"
	"strings"
	"time"
)

const (
	serverPort = 1080        // 服务器端口
	host       = "127.0.0.1" // 改为本地地址
	userName   = "test"
	password   = "test"
)

func main() {
	// 创建测试客户端
	testClient := NewTestClient(host, serverPort, userName, password)
	// 运行所有测试
	testClient.RunAllTests()
}

// TestClient SOCKS5测试客户端
type TestClient struct {
	serverAddr string
	serverPort uint16
	username   string
	password   string
}

// NewTestClient 创建新的测试客户端
func NewTestClient(serverAddr string, serverPort uint16, username, password string) *TestClient {
	return &TestClient{
		serverAddr: serverAddr,
		serverPort: serverPort,
		username:   username,
		password:   password,
	}
}

// TestBasicConnection 测试基本连接
func (tc *TestClient) TestBasicConnection() error {
	fmt.Println("=== 测试基本连接 ===")

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", tc.serverAddr, tc.serverPort))
	if err != nil {
		return fmt.Errorf("连接服务器失败: %v", err)
	}
	defer conn.Close()

	fmt.Printf("✓ 成功连接到服务器 %s:%d\n", tc.serverAddr, tc.serverPort)
	return nil
}

// TestAuthentication 测试认证功能
func (tc *TestClient) TestAuthentication() error {
	fmt.Println("=== 测试认证功能 ===")

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", tc.serverAddr, tc.serverPort))
	if err != nil {
		return fmt.Errorf("连接服务器失败: %v", err)
	}
	defer conn.Close()

	// 发送认证方法协商
	authMethods := []byte{socks5.Version, 0x02, socks5.NoAuthenticationRequired, socks5.AccountPasswordAuthentication}
	_, err = conn.Write(authMethods)
	if err != nil {
		return fmt.Errorf("发送认证方法失败: %v", err)
	}

	// 读取服务器响应
	response := make([]byte, 2)
	_, err = conn.Read(response)
	if err != nil {
		return fmt.Errorf("读取认证响应失败: %v", err)
	}

	if response[0] != socks5.Version {
		return fmt.Errorf("协议版本不匹配: 期望 %d, 实际 %d", socks5.Version, response[0])
	}

	fmt.Printf("✓ 认证方法协商成功，服务器选择方法: %d\n", response[1])

	// 如果服务器选择用户名密码认证
	if response[1] == socks5.AccountPasswordAuthentication {
		// 发送用户名密码
		authData := []byte{0x01, byte(len(tc.username))}
		authData = append(authData, []byte(tc.username)...)
		authData = append(authData, byte(len(tc.password)))
		authData = append(authData, []byte(tc.password)...)

		_, err = conn.Write(authData)
		if err != nil {
			return fmt.Errorf("发送认证数据失败: %v", err)
		}

		// 读取认证结果
		authResponse := make([]byte, 2)
		_, err = conn.Read(authResponse)
		if err != nil {
			return fmt.Errorf("读取认证结果失败: %v", err)
		}

		if authResponse[1] != 0x00 {
			return fmt.Errorf("认证失败: %d", authResponse[1])
		}

		fmt.Println("✓ 用户名密码认证成功")
	}

	return nil
}

// TestConnectCommand 测试CONNECT命令
func (tc *TestClient) TestConnectCommand(targetHost string, targetPort uint16) error {
	fmt.Printf("=== 测试CONNECT命令 (%s:%d) ===\n", targetHost, targetPort)

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", tc.serverAddr, tc.serverPort))
	if err != nil {
		return fmt.Errorf("连接服务器失败: %v", err)
	}
	defer conn.Close()

	// 执行认证
	if err := tc.performAuthentication(conn); err != nil {
		return err
	}

	// 发送CONNECT请求
	request, err := tc.buildConnectRequest(targetHost, targetPort)
	if err != nil {
		return fmt.Errorf("构建CONNECT请求失败: %v", err)
	}

	_, err = conn.Write(request)
	if err != nil {
		return fmt.Errorf("发送CONNECT请求失败: %v", err)
	}

	// 读取响应
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		return fmt.Errorf("读取CONNECT响应失败: %v", err)
	}

	response = response[:n]

	if len(response) < 4 {
		return fmt.Errorf("响应长度不足")
	}

	if response[1] != 0x00 {
		return fmt.Errorf("CONNECT请求失败，状态码: %d", response[1])
	}

	fmt.Printf("✓ CONNECT命令成功，绑定地址: %s\n", tc.parseBindAddress(response))
	return nil
}

// TestHttpProxy 测试HTTP代理功能
func (tc *TestClient) TestHttpProxy() error {
	fmt.Println("=== 测试HTTP代理功能 ===")

	client := &socks5.Client{
		Host:     tc.serverAddr,
		Port:     tc.serverPort,
		UserName: tc.username,
		Password: tc.password,
	}

	httpClient := client.GetHttpProxyClient()

	// 设置超时
	httpClient.Timeout = 10 * time.Second

	// 测试HTTP请求
	resp, err := httpClient.Get("http://httpbin.org/ip")
	if err != nil {
		return fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	fmt.Printf("✓ HTTP代理测试成功，响应长度: %d\n", len(body))
	fmt.Printf("响应内容: %s\n", string(body))

	return nil
}

// TestTcpProxy 测试TCP代理功能
func (tc *TestClient) TestTcpProxy(targetHost string, targetPort uint16) error {
	fmt.Printf("=== 测试TCP代理功能 (%s:%d) ===\n", targetHost, targetPort)

	client := &socks5.Client{
		Host:     tc.serverAddr,
		Port:     tc.serverPort,
		UserName: tc.username,
		Password: tc.password,
	}

	conn, err := client.TcpProxy(targetHost, targetPort)
	if err != nil {
		return fmt.Errorf("TCP代理连接失败: %v", err)
	}
	defer conn.Close()

	// 发送HTTP请求
	request := fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\n\r\n", targetHost)
	_, err = conn.Write([]byte(request))
	if err != nil {
		return fmt.Errorf("发送HTTP请求失败: %v", err)
	}

	// 读取响应
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	fmt.Printf("✓ TCP代理测试成功，响应: %s", response)
	return nil
}

// TestUdpProxy 测试UDP代理功能
func (tc *TestClient) TestUdpProxy(targetHost string, targetPort uint16) error {
	fmt.Printf("=== 测试UDP代理功能 (%s:%d) ===\n", targetHost, targetPort)

	client := &socks5.Client{
		Host:     tc.serverAddr,
		Port:     tc.serverPort,
		UserName: tc.username,
		Password: tc.password,
	}

	udpProxy, err := client.UdpProxy(targetHost, targetPort)
	if err != nil {
		return fmt.Errorf("UDP代理连接失败: %v", err)
	}
	defer udpProxy.Conn.Close()

	// 发送简单的UDP数据包
	testData := []byte("Hello UDP Proxy!")
	_, err = udpProxy.Conn.Write(testData)
	if err != nil {
		return fmt.Errorf("发送UDP数据失败: %v", err)
	}

	fmt.Printf("✓ UDP代理测试成功，发送数据长度: %d\n", len(testData))
	return nil
}

// TestPerformance 性能测试
func (tc *TestClient) TestPerformance() error {
	fmt.Println("=== 性能测试 ===")

	client := &socks5.Client{
		Host:     tc.serverAddr,
		Port:     tc.serverPort,
		UserName: tc.username,
		Password: tc.password,
	}

	startTime := time.Now()

	// 测试并发连接
	const numConnections = 10
	successCount := 0

	for i := 0; i < numConnections; i++ {
		conn, err := client.TcpProxy("example.com", 80)
		if err == nil {
			conn.Close()
			successCount++
		}
	}

	duration := time.Since(startTime)

	fmt.Printf("✓ 性能测试完成\n")
	fmt.Printf("  并发连接数: %d\n", numConnections)
	fmt.Printf("  成功连接数: %d\n", successCount)
	fmt.Printf("  成功率: %.2f%%\n", float64(successCount)/float64(numConnections)*100)
	fmt.Printf("  总耗时: %v\n", duration)
	fmt.Printf("  平均耗时: %v\n", duration/time.Duration(numConnections))

	return nil
}

// 辅助方法

// performAuthentication 执行认证流程
func (tc *TestClient) performAuthentication(conn net.Conn) error {
	// 发送认证方法协商
	authMethods := []byte{socks5.Version, 0x02, socks5.NoAuthenticationRequired, socks5.AccountPasswordAuthentication}
	_, err := conn.Write(authMethods)
	if err != nil {
		return fmt.Errorf("发送认证方法失败: %v", err)
	}

	// 读取服务器响应
	response := make([]byte, 2)
	_, err = conn.Read(response)
	if err != nil {
		return fmt.Errorf("读取认证响应失败: %v", err)
	}

	if response[0] != socks5.Version {
		return fmt.Errorf("协议版本不匹配")
	}

	// 如果服务器选择用户名密码认证
	if response[1] == socks5.AccountPasswordAuthentication {
		// 发送用户名密码
		authData := []byte{0x01, byte(len(tc.username))}
		authData = append(authData, []byte(tc.username)...)
		authData = append(authData, byte(len(tc.password)))
		authData = append(authData, []byte(tc.password)...)

		_, err = conn.Write(authData)
		if err != nil {
			return fmt.Errorf("发送认证数据失败: %v", err)
		}

		// 读取认证结果
		authResponse := make([]byte, 2)
		_, err = conn.Read(authResponse)
		if err != nil {
			return fmt.Errorf("读取认证结果失败: %v", err)
		}

		if authResponse[1] != 0x00 {
			return fmt.Errorf("认证失败")
		}
	}

	return nil
}

// buildConnectRequest 构建CONNECT请求
func (tc *TestClient) buildConnectRequest(host string, port uint16) ([]byte, error) {
	var request bytes.Buffer

	// 版本号、命令、保留字段
	request.Write([]byte{socks5.Version, socks5.Connect, 0x00})

	// 地址类型和地址
	if ip := net.ParseIP(host); ip != nil {
		if ip.To4() != nil {
			// IPv4
			request.WriteByte(socks5.IPv4)
			request.Write(ip.To4())
		} else {
			// IPv6
			request.WriteByte(socks5.IPv6)
			request.Write(ip.To16())
		}
	} else {
		// 域名
		request.WriteByte(socks5.Domain)
		request.WriteByte(byte(len(host)))
		request.Write([]byte(host))
	}

	// 端口
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, port)
	request.Write(portBytes)

	return request.Bytes(), nil
}

// parseBindAddress 解析绑定地址
func (tc *TestClient) parseBindAddress(response []byte) string {
	if len(response) < 4 {
		return "未知"
	}

	addrType := response[3]
	var addr string

	switch addrType {
	case socks5.IPv4:
		if len(response) >= 10 {
			ip := net.IP(response[4:8])
			port := binary.BigEndian.Uint16(response[8:10])
			addr = fmt.Sprintf("%s:%d", ip.String(), port)
		}
	case socks5.IPv6:
		if len(response) >= 22 {
			ip := net.IP(response[4:20])
			port := binary.BigEndian.Uint16(response[20:22])
			addr = fmt.Sprintf("%s:%d", ip.String(), port)
		}
	case socks5.Domain:
		if len(response) >= 7 {
			domainLen := int(response[4])
			if len(response) >= 7+domainLen {
				domain := string(response[5 : 5+domainLen])
				port := binary.BigEndian.Uint16(response[5+domainLen : 7+domainLen])
				addr = fmt.Sprintf("%s:%d", domain, port)
			}
		}
	}

	return addr
}

// RunAllTests 运行所有测试
func (tc *TestClient) RunAllTests() {
	fmt.Println("🚀 开始SOCKS5代理服务器测试")
	fmt.Println(strings.Repeat("=", 50))

	tests := []struct {
		name string
		test func() error
	}{
		{"基本连接测试", tc.TestBasicConnection},
		{"认证功能测试", tc.TestAuthentication},
		{"CONNECT命令测试", func() error { return tc.TestConnectCommand("example.com", 80) }},
		{"HTTP代理测试", tc.TestHttpProxy},
		{"TCP代理测试", func() error { return tc.TestTcpProxy("example.com", 80) }},
		{"UDP代理测试", func() error { return tc.TestUdpProxy("8.8.8.8", 53) }},
		{"性能测试", tc.TestPerformance},
	}

	passed := 0
	total := len(tests)

	for _, test := range tests {
		fmt.Printf("\n📋 %s\n", test.name)
		if err := test.test(); err != nil {
			fmt.Printf("❌ %s 失败: %v\n", test.name, err)
		} else {
			fmt.Printf("✅ %s 通过\n", test.name)
			passed++
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Printf("📊 测试结果: %d/%d 通过\n", passed, total)
	if passed == total {
		fmt.Println("🎉 所有测试通过！")
	} else {
		fmt.Printf("⚠️  %d 个测试失败\n", total-passed)
	}
}
