package util

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"time"
)

func ThisIpv4Address() (string, error) {
	address, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range address {
		// 检查IP地址是否为回环地址
		if ip, ok := address.(*net.IPNet); ok && !ip.IP.IsLoopback() {
			if ip.IP.To4() != nil {
				return ip.IP.String(), nil
			}
		}
	}
	return "127.0.0.1", nil
}

func RandomNumber(min, max, size int) int {
	rand.Seed(time.Now().UnixNano()) // 使用当前时间戳作为种子
	num := rand.Intn(size)
	if min != 0 || max != 0 {
		if num < min || num > max {
			return RandomNumber(min, max, size)
		}
	}
	return num
}

func ConsoleJson(data interface{}, tips ...string) {
	b, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		fmt.Println(err)
	}
	_tips := "------------------------"
	if len(tips) > 0 {
		_tips = tips[0]
	}
	fmt.Printf("%v\n", _tips)
	fmt.Printf("%v\n", string(b))
	fmt.Printf("%v\n", _tips)
}
