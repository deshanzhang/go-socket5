package socks5

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var ClientServer *Client

// Client SOCKS5客户端结构体
type Client struct {
	Host     string `json:"host"`     // SOCKS5服务器地址
	Port     uint16 `json:"port"`     // SOCKS5服务器端口
	UserName string `json:"username"` // 用户名（可选）
	Password string `json:"password"` // 密码（可选）
}

// AuthPackage 用于组织认证方法的结构体
type AuthPackage struct {
	methods []uint8 // 支持的认证方法列表
}

// addMethod 添加认证方法到认证包中
func (a *AuthPackage) addMethod(method uint8) {
	a.methods = append(a.methods, method)
}

// toData 将认证包转换为字节数据
func (a *AuthPackage) toData() []byte {
	data := []byte{Version, uint8(len(a.methods))}
	data = append(data, a.methods...)
	return data
}

// conn 建立与SOCKS5服务器的连接并进行认证
func (c *Client) conn() (net.Conn, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port))
	if err != nil {
		return nil, err
	}
	err = c.auth(conn)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// auth 执行SOCKS5认证流程
func (c *Client) auth(conn net.Conn) error {
	log.Printf("%s 开始认证，用户名: %s, 密码: %s", LogPrefixClient, c.UserName, c.Password)

	// 组织发送支持的认证方法
	var methods []uint8

	// 如果提供了用户名和密码，优先支持用户名密码认证
	if c.UserName != "" && c.Password != "" {
		methods = append(methods, AccountPasswordAuthentication)
		// 如果有用户名密码，也支持无认证作为备选
		methods = append(methods, NoAuthenticationRequired)
	} else {
		// 如果没有提供用户名密码，只支持无认证
		methods = append(methods, NoAuthenticationRequired)
	}

	// 构建认证请求
	authData := make([]byte, 0, 2+len(methods))
	authData = append(authData, Version, uint8(len(methods)))
	authData = append(authData, methods...)

	log.Printf("%s 发送认证方法: %x", LogPrefixClient, authData)
	_, err := conn.Write(authData)
	if err != nil {
		log.Printf("%s 发送认证方法失败: %v", LogPrefixClient, err)
		return err
	}

	// 读取服务器认证方法选择响应
	data := make([]byte, 2)
	l, err := conn.Read(data)
	if err != nil {
		log.Printf("%s 读取认证响应失败: %v", LogPrefixClient, err)
		return err
	}
	if l != 2 {
		log.Printf("%s 认证响应长度错误: %d", LogPrefixClient, l)
		return errors.New("返回数据有误非两个字节")
	}

	log.Printf("%s 收到认证响应: %x", LogPrefixClient, data)
	if data[0] != Version {
		log.Printf("%s 协议版本不匹配: %x", LogPrefixClient, data[0])
		return errors.New("当前协议Socks5与服务端协议不匹配")
	}

	// 处理服务器选择的认证方法
	switch data[1] {
	case NoAuthenticationRequired:
		log.Printf("%s 服务器选择无需认证", LogPrefixClient)
		return nil

	case AccountPasswordAuthentication:
		log.Printf("%s 服务器选择用户名密码认证", LogPrefixClient)
		if c.UserName == "" || c.Password == "" {
			log.Printf("%s 服务器要求用户名密码认证，但客户端未提供有效凭据", LogPrefixClient)
			return errors.New("服务器要求用户名密码认证，但客户端未提供有效凭据")
		}

		// 发送用户名密码认证数据
		buffer := bytes.Buffer{}
		buffer.WriteByte(0x01)                  // 认证子协议版本
		buffer.WriteByte(byte(len(c.UserName))) // 用户名长度
		buffer.WriteString(c.UserName)          // 用户名
		buffer.WriteByte(byte(len(c.Password))) // 密码长度
		buffer.WriteString(c.Password)          // 密码

		authData := buffer.Bytes()
		log.Printf("%s 发送用户名密码: %x", LogPrefixClient, authData)

		_, err = conn.Write(authData)
		if err != nil {
			log.Printf("%s 发送用户名密码失败: %v", LogPrefixClient, err)
			return err
		}

		// 读取认证结果
		l, err = conn.Read(data)
		if err != nil {
			log.Printf("%s 读取认证结果失败: %v", LogPrefixClient, err)
			return err
		}
		if l != 2 {
			log.Printf("%s 认证结果长度错误: %d", LogPrefixClient, l)
			return errors.New("返回数据有误非两个字节")
		}

		log.Printf("%s 收到认证结果: %x", LogPrefixClient, data)
		if data[0] != 0x01 {
			log.Printf("%s 认证协议不匹配: %x", LogPrefixClient, data[0])
			return errors.New("当前认证协议Socks5与服务端协议不匹配")
		}
		if data[1] > 0 {
			log.Printf("%s 用户名密码认证失败，错误码: %x", LogPrefixClient, data[1])
			return fmt.Errorf("用户名密码认证失败，错误码: %x", data[1])
		}

		log.Printf("%s 认证成功", LogPrefixClient)
		return nil

	case 0xFF:
		log.Printf("%s 服务器拒绝所有认证方法", LogPrefixClient)
		return errors.New("服务器拒绝所有认证方法")

	default:
		log.Printf("%s 服务器选择不支持的认证方法: %x", LogPrefixClient, data[1])
		return errors.New("服务器选择不支持的认证方法")
	}
}

// UdpProxy UDP代理结构体
type UdpProxy struct {
	Conn    net.Conn     // TCP控制连接
	UdpConn *net.UDPConn // UDP数据连接
	Host    string       // 目标主机
	Port    uint16       // 目标端口
	client  *Client      // 客户端引用
}

// portToBytes 将端口号转换为字节数组（大端序）
func portToBytes(port uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, port)
	return b
}

// requisition 发送SOCKS5请求并处理响应
func (c *Client) requisition(conn net.Conn, host string, port uint16, cmd uint8) (net.Conn, error) {
	log.Printf("%s 开始发送SOCKS5请求: cmd=%d, host=%s, port=%d", LogPrefixClient, cmd, host, port)
	var Type byte
	var addr []byte

	if cmd == UDP && host == "0.0.0.0" && port == 0 {
		// 对于UDP命令，使用IPv4地址类型
		Type = IPv4
		addr = net.ParseIP("0.0.0.0").To4()
		log.Printf("%s UDP命令使用IPv4地址: %x", LogPrefixClient, addr)
	} else {
		Type, addr, _ = addressResolution(fmt.Sprintf("%s:%d", host, port))
		log.Printf("%s 地址解析结果: Type=%d, addr=%x", LogPrefixClient, Type, addr)
		if Type == 0 {
			log.Printf("%s 地址解析失败", LogPrefixClient)
			return nil, errors.New("addressResolution 失败")
		}
	}

	buffer := bytes.Buffer{}
	buffer.Write([]byte{Version, cmd, Zero, Type})
	buffer.Write(addr)
	buffer.Write(portToBytes(port))
	requestData := buffer.Bytes()
	log.Printf("%s 发送SOCKS5请求: %x", LogPrefixClient, requestData)
	_, err := conn.Write(requestData)
	if err != nil {
		log.Printf("%s 发送SOCKS5请求失败: %v", LogPrefixClient, err)
		return nil, err
	}
	buf := make([]byte, 1024)
	read, err := conn.Read(buf)
	if err != nil {
		log.Printf("%s 读取SOCKS5响应失败: %v", LogPrefixClient, err)
		return nil, err
	}
	rdata := buf[:read]
	log.Printf("%s 收到SOCKS5响应: %x", LogPrefixClient, rdata)
	if len(rdata) < 2 || rdata[1] != Zero {
		log.Printf("%s SOCKS5响应错误: %x", LogPrefixClient, rdata)
		return nil, errors.New(fmt.Sprintf("请求错误:%x", rdata[1]))
	}

	// 对于CONNECT命令，需要读取完整的响应（包括绑定地址）
	if cmd == Connect {
		log.Printf("%s 处理CONNECT响应", LogPrefixClient)
		// 检查响应长度，如果只有4字节，需要读取更多数据
		if len(rdata) == 4 {
			log.Printf("%s CONNECT响应只有4字节，需要读取更多数据", LogPrefixClient)
			// 需要读取额外的地址信息
			additionalData := make([]byte, 1024)
			additionalRead, err := conn.Read(additionalData)
			if err != nil {
				log.Printf("%s 读取额外地址信息失败: %v", LogPrefixClient, err)
				return nil, err
			}
			rdata = append(rdata, additionalData[:additionalRead]...)
			log.Printf("%s 完整CONNECT响应: %x", LogPrefixClient, rdata)
		}
	}

	return conn, nil
}

// udp 建立UDP代理连接
func (c *Client) udp(conn net.Conn, host string, port uint16) (net.Conn, error) {
	// 对于UDP命令，我们使用"0.0.0.0:0"作为目标地址
	// 因为SOCKS5服务器会返回实际的绑定地址
	return c.requisition(conn, "0.0.0.0", 0, UDP)
}

// bind 执行BIND命令
func (c *Client) bind(conn net.Conn, host string, port uint16) error {
	_, err := c.requisition(conn, host, port, Bind)
	return err
}

// tcp 执行CONNECT命令
func (c *Client) tcp(conn net.Conn, host string, port uint16) error {
	_, err := c.requisition(conn, host, port, Connect)
	return err
}

// TcpProxy TCP代理
func (c *Client) TcpProxy(host string, port uint16) (net.Conn, error) {
	conn, err := c.conn()
	if err != nil {
		return nil, err
	}
	return c.requisition(conn, host, port, Connect)
}

// GetHttpProxyClient 获取配置了SOCKS5代理的HTTP客户端
func (c *Client) GetHttpProxyClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, portStr, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				port, err := strconv.ParseUint(portStr, 10, 16)
				if err != nil {
					return nil, err
				}
				return c.TcpProxy(host, uint16(port))
			},
		},
	}
}

// GetHttpProxyClientSpecify 获取自定义配置的HTTP代理客户端
func (c *Client) GetHttpProxyClientSpecify(transport *http.Transport, jar http.CookieJar, CheckRedirect func(req *http.Request, via []*http.Request) error, Timeout time.Duration) *http.Client {
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		split := strings.Split(addr, ":")
		if len(split) < 2 {
			return c.TcpProxy(split[0], 80)
		}
		port, err := strconv.Atoi(split[1])
		if err != nil {
			return nil, err
		}
		return c.TcpProxy(split[0], uint16(port))
	}
	return &http.Client{Transport: transport, Jar: jar, CheckRedirect: CheckRedirect, Timeout: Timeout}
}

// UdpProxy UDP代理
func (c *Client) UdpProxy(host string, port uint16) (*UdpProxy, error) {
	conn, err := c.conn()
	if err != nil {
		return nil, err
	}
	_, err = c.requisition(conn, host, port, UDP)
	if err != nil {
		return nil, err
	}
	return &UdpProxy{
		Conn:   conn,
		Host:   host,
		Port:   port,
		client: c,
	}, nil
}

// Close 关闭UDP代理
func (u *UdpProxy) Close() error {
	if u.UdpConn != nil {
		u.UdpConn.Close()
	}
	if u.Conn != nil {
		u.Conn.Close()
	}
	return nil
}

// SendUdpPacket 发送UDP数据包
func (u *UdpProxy) SendUdpPacket(data []byte) error {
	// 构建SOCKS5 UDP请求头
	var Type byte
	var addr []byte

	// 解析目标地址
	Type, addr, _ = addressResolution(fmt.Sprintf("%s:%d", u.Host, u.Port))
	if Type == 0 {
		// 如果地址解析失败，尝试手动解析
		if ip := net.ParseIP(u.Host); ip != nil {
			if ip.To4() != nil {
				Type = IPv4
				addr = ip.To4()
			} else {
				Type = IPv6
				addr = ip.To16()
			}
		} else {
			Type = Domain
			addr = append([]byte{byte(len(u.Host))}, []byte(u.Host)...)
		}
	}

	// 构建UDP请求包
	buffer := bytes.Buffer{}
	buffer.Write([]byte{0x00, 0x00, 0x00}) // RSV + FRAG
	buffer.WriteByte(Type)
	buffer.Write(addr)
	buffer.Write(portToBytes(u.Port))
	buffer.Write(data)

	// 发送UDP数据包
	_, err := u.UdpConn.Write(buffer.Bytes())
	return err
}

// ReceiveUdpPacket 接收UDP数据包
func (u *UdpProxy) ReceiveUdpPacket() ([]byte, error) {
	buffer := make([]byte, 4096)
	n, err := u.UdpConn.Read(buffer)
	if err != nil {
		return nil, err
	}

	// 解析SOCKS5 UDP响应头
	if n < 10 {
		return nil, errors.New("UDP响应数据长度不足")
	}

	// 跳过RSV + FRAG + ATYP
	headerLen := 3 + 1
	addrType := buffer[3]

	var addrLen int
	switch addrType {
	case IPv4:
		addrLen = 4
	case Domain:
		addrLen = 1 + int(buffer[4])
	case IPv6:
		addrLen = 16
	default:
		return nil, errors.New("不支持的地址类型")
	}

	// 计算头部总长度：RSV + FRAG + ATYP + ADDR + PORT
	headerLen += addrLen + 2

	if n < headerLen {
		return nil, errors.New("UDP响应头部长度不足")
	}

	// 返回数据部分
	return buffer[headerLen:n], nil
}

// SendAndReceiveUdpPacket 发送UDP数据包并接收响应
func (u *UdpProxy) SendAndReceiveUdpPacket(data []byte) ([]byte, error) {
	err := u.SendUdpPacket(data)
	if err != nil {
		return nil, err
	}

	// 设置超时
	u.UdpConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	defer u.UdpConn.SetReadDeadline(time.Time{})

	return u.ReceiveUdpPacket()
}
