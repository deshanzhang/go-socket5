package server

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Socks5Config struct {
	Listen     string `yaml:"listen"`
	HttpServer string `yaml:"http_server"`
	AuthList   []int  `yaml:"auth_list"`
}

// LoadConfig 读取配置文件
func LoadConfig(path string) (*Socks5Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cfg Socks5Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
