package utils

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func ReadIPListFromFile(f string) ([]string, error) {
	file, err := os.Open(f)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	ipList := make([]string, 0, 128)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ip := strings.TrimSpace(scanner.Text())
		if len(ip) == 0 {
			continue
		}
		if strings.Contains(ip, "/") {
			_, _, err := net.ParseCIDR(ip)
			if err != nil {
				return nil, fmt.Errorf("scan ip failed, cidr:%s, err:%w", ip, err)
			}
		} else { //
			if parsed := net.ParseIP(ip); parsed == nil {
				return nil, fmt.Errorf("scan ip failed, ip:%s, err:%w", ip, err)
			}
		}

		ipList = append(ipList, ip)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return ipList, err
}
