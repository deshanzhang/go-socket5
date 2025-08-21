package socks5

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
)

// addressResolutionFormByteArray 从字节数组解析地址
// 根据地址类型解析字节数组，返回格式化的地址字符串
func addressResolutionFormByteArray(data []byte, addrType byte) (string, error) {
	log.Printf("%s 解析地址: addrType=%d, data=%x", LogPrefixUtils, addrType, data)
	switch addrType {
	case IPv4:
		if len(data) < 4 {
			return "", fmt.Errorf("IPv4地址长度不足")
		}
		ip := net.IP(data[:4])
		if len(data) < 6 {
			return "", fmt.Errorf("IPv4地址端口长度不足")
		}
		port := binary.BigEndian.Uint16(data[4:6])
		result := fmt.Sprintf("%s:%d", ip.String(), port)
		log.Printf("%s IPv4解析结果: %s", LogPrefixUtils, result)
		return result, nil

	case Domain:
		if len(data) < 1 {
			return "", fmt.Errorf("域名长度不足")
		}
		domainLen := int(data[0])
		log.Printf("%s 域名长度: %d, 总数据长度: %d", LogPrefixUtils, domainLen, len(data))
		if len(data) < 1+domainLen+2 {
			return "", fmt.Errorf("域名地址长度不足")
		}
		domain := string(data[1 : 1+domainLen])
		port := binary.BigEndian.Uint16(data[1+domainLen : 1+domainLen+2])
		result := fmt.Sprintf("%s:%d", domain, port)
		log.Printf("%s 域名解析结果: %s", LogPrefixUtils, result)
		return result, nil

	case IPv6:
		if len(data) < 16 {
			return "", fmt.Errorf("IPv6地址长度不足")
		}
		ip := net.IP(data[:16])
		if len(data) < 18 {
			return "", fmt.Errorf("IPv6地址端口长度不足")
		}
		port := binary.BigEndian.Uint16(data[16:18])
		result := fmt.Sprintf("%s:%d", ip.String(), port)
		log.Printf("%s IPv6解析结果: %s", LogPrefixUtils, result)
		return result, nil

	default:
		return "", fmt.Errorf("不支持的地址类型: %d", addrType)
	}
}

// addressResolution 解析地址字符串，返回地址类型、主机地址和端口
// 将地址字符串解析为SOCKS5协议所需的格式
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
	return Domain, append([]byte{byte(len(host))}, []byte(host)...), uint16(port)
}

// ioCopy 在两个连接之间复制数据
// 实现双向数据转发，用于代理连接
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

// IsValidAuthMethod 检查认证方法是否有效
func IsValidAuthMethod(method uint8) bool {
	switch method {
	case NoAuthenticationRequired, AccountPasswordAuthentication:
		return true
	default:
		return false
	}
}

// GetAuthMethodName 获取认证方法名称
func GetAuthMethodName(method uint8) string {
	switch method {
	case NoAuthenticationRequired:
		return "无需认证"
	case AccountPasswordAuthentication:
		return "用户名密码认证"
	default:
		return fmt.Sprintf("未知认证方法(0x%02x)", method)
	}
}

// ValidateAuthConfig 验证认证配置的有效性
func ValidateAuthConfig(authList []uint8, userMap map[string]string) []string {
	var warnings []string

	if len(authList) == 0 {
		warnings = append(warnings, "认证方法列表为空")
		return warnings
	}

	hasPasswordAuth := false
	hasValidMethod := false

	for _, method := range authList {
		if IsValidAuthMethod(method) {
			hasValidMethod = true
			if method == AccountPasswordAuthentication {
				hasPasswordAuth = true
			}
		} else {
			warnings = append(warnings, fmt.Sprintf("无效的认证方法: 0x%02x", method))
		}
	}

	if !hasValidMethod {
		warnings = append(warnings, "没有有效的认证方法")
	}

	if hasPasswordAuth && len(userMap) == 0 {
		warnings = append(warnings, "配置了用户名密码认证但没有用户数据")
	}

	if hasPasswordAuth {
		for username, password := range userMap {
			if username == "" {
				warnings = append(warnings, "存在空用户名")
			}
			if password == "" {
				warnings = append(warnings, fmt.Sprintf("用户 '%s' 的密码为空", username))
			}
		}
	}

	return warnings
}

// ToUint8Slice 辅助函数，将[]int转[]uint8
func ToUint8Slice(arr []int) []uint8 {
	r := make([]uint8, len(arr))
	for i, v := range arr {
		r[i] = uint8(v)
	}
	return r
}
