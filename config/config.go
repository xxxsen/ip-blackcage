package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/xxxsen/common/logger"
)

type Config struct {
	BlackPortList []string `json:"black_port_list"`
	BlackListSet  string   `json:"blacklist_set"`
	LogConfig     logger.LogConfig
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
		BlackListSet: "ip-blackcage",
	}
	if err := json.Unmarshal(raw, c); err != nil {
		return nil, err
	}
	return c, nil
}
