package socks5

const (
	// Version SOCKS5 协议版本号
	Version = 0x05

	// NoAuthenticationRequired 无需认证
	NoAuthenticationRequired = 0x00
	// AccountPasswordAuthentication 用户名密码认证
	AccountPasswordAuthentication = 0x02

	// Connect CONNECT命令 - 建立TCP连接
	Connect = 0x01
	// Bind BIND命令 - 绑定端口
	Bind = 0x02
	// UDP UDP命令 - UDP转发
	UDP = 0x03

	// IPv4 IPv4地址类型
	IPv4 = 0x01
	// Domain 域名地址类型
	Domain = 0x03
	// IPv6 IPv6地址类型
	IPv6 = 0x04

	// Zero 成功状态码
	Zero = 0x00
)
