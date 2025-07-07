package server

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Socks5Config struct {
	Host      string   `yaml:"host"`
	Port      int      `yaml:"port"`
	User      string   `yaml:"user"`
	Password  string   `yaml:"password"`
	BlackList []string `yaml:"blacklist"`
	AuthList  []int    `yaml:"auth_list"`
}

type ProjectConfig struct {
	Socks5 Socks5Config `yaml:"socks5"`
	Gin    GinConfig    `yaml:"gin"`
}

// LoadConfig 读取配置文件
func LoadConfig(path string) (*ProjectConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cfg ProjectConfig
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
