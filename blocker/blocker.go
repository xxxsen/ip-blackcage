package blocker

import (
	"context"
	"fmt"
	"ip-blackcage/ipset"
	"strings"

	"github.com/coreos/go-iptables/iptables"
	"github.com/xxxsen/common/logutil"
	"go.uber.org/zap"
)

const (
	defaultBlackSet        = "ip-blackcage-blacklist-set"
	defaultWhiteSet        = "ip-blackcage-whitelist-set"
	defaultFilterTable     = "filter"
	defaultCageChain       = "ip-blackcage-chain"
	defaultDockerUserChain = "DOCKER-USER"
)

type IBlocker interface {
	Init(ctx context.Context, blackips []string, whiteips []string) error
	Destroy(ctx context.Context) error
	BanIP(ctx context.Context, ip string) error
	UnBanIP(ctx context.Context, ip string) error
	WhiteIP(ctx context.Context, ip string) error
	UnWhiteIP(ctx context.Context, ip string) error
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

func (f *defaultBlocker) getWhiteSet() string {
	return defaultWhiteSet
}

func (f *defaultBlocker) getTmpSet(n string) string {
	return n + "-tmp"
}

func (f *defaultBlocker) ensureIPSet(ctx context.Context, setname string, ips []string) error {
	tmpset := f.getTmpSet(setname)
	if err := f.set.Create(ctx, setname, ipset.SetTypeHashNet, ipset.WithExist()); err != nil {
		return fmt.Errorf("create ip set failed, err:%w", err)
	}
	if err := f.set.Destroy(ctx, tmpset, ipset.WithExist()); err != nil {
		return fmt.Errorf("destroy ip tmp set failed, err:%w", err)
	}
	if err := f.set.Create(ctx, tmpset, ipset.SetTypeHashNet, ipset.WithExist()); err != nil {
		return fmt.Errorf("create ip tmp set failed, err:%w", err)
	}
	for _, ip := range ips {
		if err := f.set.Add(ctx, tmpset, ip, ipset.WithExist()); err != nil {
			return fmt.Errorf("add ip:%s to set failed, err:%w", ip, err)
		}
	}
	if err := f.set.Swap(ctx, tmpset, setname); err != nil {
		return fmt.Errorf("swap black set failed, err:%w", err)
	}
	if err := f.set.Destroy(ctx, tmpset, ipset.WithExist()); err != nil {
		return fmt.Errorf("destroy tmp set failed, err:%w", err)
	}
	return nil
}

func (f *defaultBlocker) ensureInputChain(_ context.Context) error {
	table := defaultFilterTable
	chain := defaultCageChain
	blackset := f.getBlackSet()
	whiteset := f.getWhiteSet()
	ok, err := f.ipt.ChainExists(table, chain)
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
			name: "allow established",
			args: []string{"-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT"},
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

func (f *defaultBlocker) ensureDockerChain(_ context.Context) error {
	table := defaultFilterTable
	chain := defaultCageChain
	dockerchain := defaultDockerUserChain
	ok, err := f.ipt.ChainExists(table, dockerchain)
	if err != nil {
		return err
	}
	if !ok {
		if err := f.ipt.NewChain(table, dockerchain); err != nil {
			return err
		}
	}
	if err = f.ipt.InsertUnique(table, dockerchain, 1, "-j", chain); err != nil {
		return fmt.Errorf("append docker chain failed, err:%w", err)
	}
	return nil
}

func (f *defaultBlocker) ensureIPTable(ctx context.Context) error {
	if err := f.ensureInputChain(ctx); err != nil {
		return fmt.Errorf("ensure input chain failed, err:%w", err)
	}
	if err := f.ensureDockerChain(ctx); err != nil {
		return fmt.Errorf("ensure docker chain failed, err:%w", err)
	}
	return nil
}

func (f *defaultBlocker) Destroy(ctx context.Context) error {
	table := defaultFilterTable
	chain := defaultCageChain
	blackset := f.getBlackSet()
	whiteset := f.getWhiteSet()
	//移除docker-user链上的处理流程
	if err := f.ipt.DeleteIfExists(table, defaultDockerUserChain, "-j", chain); err != nil {
		if !strings.Contains(err.Error(), "does not exist") {
			logutil.GetLogger(ctx).Error("delete docker-user chain jump rule failed", zap.Error(err))
			return err
		}
	}
	//移除input链上的处理流程
	if err := f.ipt.DeleteIfExists(table, "INPUT", "-j", chain); err != nil {
		if !strings.Contains(err.Error(), "does not exist") {
			logutil.GetLogger(ctx).Error("delete input chain jump rule failed", zap.Error(err))
			return err
		}
		//不存在的错误直接忽略
	}
	if err := f.ipt.ClearAndDeleteChain(table, chain); err != nil {
		return fmt.Errorf("clean and delete chain failed, err:%w", err)
	}
	_ = f.set.Destroy(ctx, blackset, ipset.WithExist())
	_ = f.set.Destroy(ctx, whiteset, ipset.WithExist())
	return nil
}

func (f *defaultBlocker) Init(ctx context.Context, blackIps []string, whiteIps []string) error {
	if err := f.Destroy(ctx); err != nil { //先进行预处理
		return fmt.Errorf("destroy before init failed, err:%w", err)
	}
	if err := f.ensureIPSet(ctx, f.getWhiteSet(), whiteIps); err != nil {
		return fmt.Errorf("ensure white ip set failed, err:%w", err)
	}
	if err := f.ensureIPSet(ctx, f.getBlackSet(), blackIps); err != nil {
		return fmt.Errorf("ensure black ip set failed, err:%w", err)
	}
	if err := f.ensureIPTable(ctx); err != nil {
		return err
	}
	return nil
}

func (f *defaultBlocker) BanIP(ctx context.Context, ip string) error {
	return f.set.Add(ctx, f.getBlackSet(), ip, ipset.WithExist())
}

func (f *defaultBlocker) UnBanIP(ctx context.Context, ip string) error {
	return f.set.Del(ctx, f.getBlackSet(), ip)
}

func (f *defaultBlocker) WhiteIP(ctx context.Context, ip string) error {
	return f.set.Add(ctx, f.getWhiteSet(), ip, ipset.WithExist())
}

func (f *defaultBlocker) UnWhiteIP(ctx context.Context, ip string) error {
	return f.set.Del(ctx, f.getWhiteSet(), ip)
}
