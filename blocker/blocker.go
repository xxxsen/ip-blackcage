package blocker

import (
	"context"
	"fmt"
	"ip-blackcage/ipset"

	"github.com/coreos/go-iptables/iptables"
)

const (
	defaultBlackSet  = "ip-blackcage-blacklist-set"
	defaultWhiteSet  = "ip-blackcage-whitelist-set"
	defaultCageChain = "ip-blackcage-chain"
)

type IBlocker interface {
	Init(ctx context.Context, ips []string) error
	BanIP(ctx context.Context, ip string) error
	UnBanIP(ctx context.Context, ip string) error
}

type defaultBlocker struct {
	ipt *iptables.IPTables
	set *ipset.IPSet
}

func NewBlocker() (IBlocker, error) {
	set, err := ipset.New()
	if err != nil {
		return nil, err
	}
	ipt, err := iptables.New()
	if err != nil {
		return nil, err
	}
	return &defaultBlocker{
		ipt: ipt,
		set: set,
	}, nil
}

func (f *defaultBlocker) getBlackSet() string {
	return defaultBlackSet
}

func (f *defaultBlocker) getCageChain() string {
	return defaultCageChain
}

func (f *defaultBlocker) getWhiteSet() string {
	return defaultWhiteSet
}

func (f *defaultBlocker) getTmpSet(n string) string {
	return n + "-tmp"
}

func (f *defaultBlocker) ensureIPSet(ctx context.Context, ips []string) error {
	blackset := f.getBlackSet()
	whiteset := f.getWhiteSet()
	tmpset := f.getTmpSet(blackset)
	if err := f.set.Create(ctx, blackset, ipset.HashNetType, ipset.WithExist()); err != nil {
		return fmt.Errorf("create black set failed, err:%w", err)
	}
	if err := f.set.Create(ctx, whiteset, ipset.HashNetType, ipset.WithExist()); err != nil {
		return fmt.Errorf("create white set failed, err:%w", err)
	}
	if err := f.set.Destroy(ctx, tmpset, ipset.WithExist()); err != nil {
		return fmt.Errorf("destroy black tmp set failed, err:%w", err)
	}
	if err := f.set.Create(ctx, tmpset, ipset.HashNetType, ipset.WithExist()); err != nil {
		return fmt.Errorf("create black tmp set failed, err:%w", err)
	}
	for _, ip := range ips {
		if err := f.set.Add(ctx, tmpset, ip); err != nil {
			return fmt.Errorf("add ip:%s to set failed, err:%w", ip, err)
		}
	}
	if err := f.set.Swap(ctx, tmpset, blackset); err != nil {
		return fmt.Errorf("swap black set failed, err:%w", err)
	}
	if err := f.set.Destroy(ctx, tmpset, ipset.WithExist()); err != nil {
		return fmt.Errorf("destroy tmp set failed, err:%w", err)
	}
	return nil
}

func (f *defaultBlocker) ensureIPTable(ctx context.Context) error {
	table := "filter"
	chain := f.getCageChain()
	blackset := f.getBlackSet()
	whiteset := f.getWhiteSet()
	ok, err := f.ipt.Exists(table, chain)
	if err != nil {
		return err
	}
	if !ok {
		if err := f.ipt.NewChain(table, chain); err != nil {
			return err
		}
	}

	rules := []struct {
		name string
		args []string
	}{
		{
			name: "skip whitelist",
			args: []string{"-m", "set", "--match-set", whiteset, "src", "-j", "RETURN"},
		},
		{
			name: "drop traffic",
			args: []string{"-m", "set", "--match-set", blackset, "src", "-j", "DROP"},
		},
		{
			name: "return origin",
			args: []string{"-j", "RETURN"},
		},
	}
	for _, rule := range rules {
		if err := f.ipt.AppendUnique(table, chain, rule.args...); err != nil {
			return fmt.Errorf("create rule:%s failed, err:%w", rule.name, err)
		}
	}
	if err = f.ipt.InsertUnique(table, "INPUT", 1, "-j", chain); err != nil {
		return fmt.Errorf("inset to input chains failed, err:%w", err)
	}
	return nil
}

func (f *defaultBlocker) Init(ctx context.Context, ips []string) error {
	if err := f.ensureIPSet(ctx, ips); err != nil {
		return err
	}
	if err := f.ensureIPTable(ctx); err != nil {
		return err
	}
	return nil
}

func (f *defaultBlocker) BanIP(ctx context.Context, ip string) error {
	return f.set.Add(ctx, f.getBlackSet(), ip)
}

func (f *defaultBlocker) UnBanIP(ctx context.Context, ip string) error {
	return f.set.Del(ctx, f.getBlackSet(), ip)
}
