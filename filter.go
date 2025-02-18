package ipblackcage

import (
	"fmt"
	"net"
)

type ipNetFilter struct {
	filterList []*net.IPNet
}

func newIPNetFilter() *ipNetFilter {
	return &ipNetFilter{}
}

func (f *ipNetFilter) AddRule(ips ...string) error {
	for _, ip := range ips {
		_, cidr, err := net.ParseCIDR(ip)
		if err != nil {
			return err
		}
		f.filterList = append(f.filterList, cidr)
	}
	return nil
}

func (f *ipNetFilter) IsContains(ip string) (bool, error) {
	nip := net.ParseIP(ip)
	if nip == nil {
		return false, fmt.Errorf("parse ip failed, ipdata:%s", ip)
	}
	for _, filter := range f.filterList {
		if filter.Contains(nip) {
			return true, nil
		}
	}
	return false, nil
}
