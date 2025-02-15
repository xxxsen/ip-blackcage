package blocker

import (
	"context"
	"fmt"
	"os/exec"
)

var (
	defaultCheckList = []string{"iptables", "ipset"}
)

type iptable struct {
	set string
}

func NewIPTable(set string) (IBlocker, error) {
	for _, cmd := range defaultCheckList {
		if _, err := exec.LookPath(cmd); err != nil {
			return nil, fmt.Errorf("check cmd:%s failed, err:%w", cmd, err)
		}
	}
	if len(set) == 0 {
		return nil, fmt.Errorf("invalid set name")
	}
	return &iptable{set: set}, nil
}

func (f *iptable) Init(ctx context.Context, ips []string) error {
	//TODO: finish it
	panic("impl it")
}

func (f *iptable) BanIP(ctx context.Context, ip string) error {
	// TODO: finish it
	panic("impl it")
}

func (f *iptable) UnBanIP(ctx context.Context, ip string) error {
	panic("not implemented") // TODO: Implement
}
