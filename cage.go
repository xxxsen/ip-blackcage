package ipblackcage

import (
	"context"
	"fmt"

	"github.com/xxxsen/common/logutil"
	"go.uber.org/zap"
)

type IPBlackCage struct {
	c        *config
	blackMap map[string]struct{}
}

func New(opts ...Option) (*IPBlackCage, error) {
	c := applyOpts(opts...)
	if c.filter == nil {
		return nil, fmt.Errorf("no filter found")
	}
	if c.obs == nil {
		return nil, fmt.Errorf("no observer found")
	}
	return &IPBlackCage{c: c, blackMap: make(map[string]struct{}, 2048)}, nil
}

func (bc *IPBlackCage) Run(ctx context.Context) error {
	ch, err := bc.c.obs.Open(ctx)
	if err != nil {
		return err
	}
	for ev := range ch {
		ip := ev.Data().(string)
		ts := ev.Timestamp()
		ok, err := bc.addToBlackList(ctx, ip)
		if err != nil {
			logutil.GetLogger(ctx).Error("add ip to black list failed", zap.Error(err), zap.String("ip", ip))
			continue
		}
		if !ok {
			continue
		}
		logutil.GetLogger(ctx).Info("add ip to black list succ", zap.String("ip", ip), zap.Int64("ts", ts))
	}
	return nil
}

func (bc *IPBlackCage) addToBlackList(ctx context.Context, ip string) (bool, error) {
	if _, ok := bc.blackMap[ip]; ok {
		return false, nil
	}
	if err := bc.c.filter.BanIP(ctx, ip); err != nil {
		return false, err
	}
	bc.blackMap[ip] = struct{}{}
	return true, nil
}
