package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/xxxsen/common/logger"
)

type NetConfig struct {
	Interface string   `json:"interface"`
	ExitIPs   []string `json:"exit_ips"`
}

type Config struct {
	NetConfig                  NetConfig        `json:"net_config"`
	BlackPortList              []string         `json:"black_port_list"`
	DBFile                     string           `json:"db_file"`
	LogConfig                  logger.LogConfig `json:"log_config"`
	UserIPBlackListDir         string           `json:"user_ip_black_list_dir"`
	UserIPWhiteListDir         string           `json:"user_ip_white_list_dir"`
	ViewMode                   bool             `json:"view_mode"`
	BanTime                    uint64           `json:"ban_time"`
	DisableLocalNetworkProtect bool             `json:"disable_local_network_protect"`
	CageSize                   uint64           `json:"cage_size"`
}

func (c *Config) DecodePortList() ([]uint16, error) {
	m := make(map[uint16]struct{})
	for _, pstr := range c.BlackPortList {
		ports := strings.Split(pstr, "-")
		left, err := strconv.ParseUint(ports[0], 10, 64)
		if err != nil {
			return nil, err
		}
		right := left
		if len(ports) > 1 {
			right, err = strconv.ParseUint(ports[1], 10, 64)
			if err != nil {
				return nil, err
			}
		}
		if right < left {
			return nil, fmt.Errorf("invalid port range:%s", pstr)
		}
		for i := left; i <= right; i++ {
			m[uint16(i)] = struct{}{}
		}
	}
	rs := make([]uint16, 0, len(m))
	for p := range m {
		rs = append(rs, p)
	}
	return rs, nil
}

func Parse(f string) (*Config, error) {
	raw, err := os.ReadFile(f)
	if err != nil {
		return nil, err
	}
	c := &Config{
		BanTime:  3 * 30 * 86400, // 90d
		CageSize: 100000,
	}
	if err := json.Unmarshal(raw, c); err != nil {
		return nil, err
	}
	return c, nil
}
