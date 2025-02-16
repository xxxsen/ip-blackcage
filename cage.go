package ipblackcage

import (
	"context"
	"fmt"
	"ip-blackcage/model"
	"os"
	"os/signal"
	"syscall"

	"github.com/xxxsen/common/logutil"
	"go.uber.org/zap"
)

type IPBlackCage struct {
	c *config
}

func New(opts ...Option) (*IPBlackCage, error) {
	c := applyOpts(opts...)
	if c.filter == nil {
		return nil, fmt.Errorf("no filter found")
	}
	if c.obs == nil {
		return nil, fmt.Errorf("no observer found")
	}
	return &IPBlackCage{c: c}, nil
}

func (bc *IPBlackCage) initBlackList(ctx context.Context) error {
	iplist := make([]string, 0, 1024)
	cnt, err := bc.c.ipDao.ListBlackIP(ctx, 200, func(ctx context.Context, ips []*model.BlackCageTab) error {
		for _, ip := range ips {
			iplist = append(iplist, ip.IP)
		}
		return nil
	})
	if err != nil {
		return err
	}
	logutil.GetLogger(ctx).Info("read black ips from db succ", zap.Int64("cnt", cnt))
	if err := bc.c.filter.Init(ctx, iplist); err != nil {
		return err
	}
	return nil
}

func (bc *IPBlackCage) registerCleanBlackListSignal(ctx context.Context) error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		logutil.GetLogger(ctx).Info("recv stop signal, clean blocker rules", zap.Any("signal", sig.String()))
		if err := bc.c.filter.Destroy(ctx); err != nil {
			logutil.GetLogger(ctx).Error("clean blocker rules failed", zap.Error(err))
			return
		}
	}()
	return nil
}

func (bc *IPBlackCage) Run(ctx context.Context) error {
	if err := bc.registerCleanBlackListSignal(ctx); err != nil {

	}
	if err := bc.initBlackList(ctx); err != nil {
		return err
	}
	ch, err := bc.c.obs.Open(ctx)
	if err != nil {
		return err
	}
	for ev := range ch {
		evn := ev.EventType()
		ip := ev.Data().(string)
		ts := ev.Timestamp()
		ok, err := bc.addToBlackList(ctx, evn, ip, ts)
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

func (bc *IPBlackCage) addToBlackList(ctx context.Context, ev string, ip string, ts int64) (bool, error) {
	_, ok, err := bc.c.ipDao.GetBlackIP(ctx, ip)
	if err != nil {
		return false, err
	}
	if ok {
		return false, nil
	}
	if err := bc.c.filter.BanIP(ctx, ip); err != nil {
		return false, err
	}
	bc.c.ipDao.AddBlackIP(ctx, &model.BlackCageTab{
		CTime:  uint64(ts),
		MTime:  uint64(ts),
		IP:     ip,
		IPType: fmt.Sprintf("detect_by_event:%s", ev),
	})
	return true, nil
}
