package ipblackcage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/xxxsen/common/logutil"
	"go.uber.org/zap"
)

type IPBlackCage struct {
	c        *config
	lck      sync.RWMutex
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

func (bc *IPBlackCage) saveBlackIPData(ctx context.Context) error {
	bc.lck.Lock()
	ips := make([]string, 0, len(bc.blackMap))
	for ip := range bc.blackMap {
		ips = append(ips, ip)
	}
	bc.lck.Unlock()
	raw, err := json.Marshal(&ips)
	if err != nil {
		return err
	}
	tmpfile := bc.c.savefile + "-tmp"
	if err := os.WriteFile(tmpfile, raw, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmpfile, bc.c.savefile); err != nil {
		logutil.GetLogger(ctx).Error("rename tmp file to save file failed", zap.Error(err))
	}
	return nil
}

func (bc *IPBlackCage) startSaveThread(ctx context.Context) {
	if len(bc.c.savefile) == 0 {
		return
	}
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		if err := bc.saveBlackIPData(ctx); err != nil {
			logutil.GetLogger(ctx).Error("save black ip data failed", zap.Error(err))
			continue
		}
	}
}

func (bc *IPBlackCage) initBlackList(ctx context.Context) error {
	if len(bc.c.savefile) == 0 {
		return nil
	}
	data, err := os.ReadFile(bc.c.savefile)
	ips := make([]string, 0, 1024)
	if err == nil {
		if err := json.Unmarshal(data, &ips); err != nil {
			return err
		}
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	for _, ip := range ips {
		bc.blackMap[ip] = struct{}{}
	}
	if err := bc.c.filter.Init(ctx, ips); err != nil {
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
	go bc.startSaveThread(ctx)
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
	bc.lck.RLock()
	defer bc.lck.RUnlock()
	if _, ok := bc.blackMap[ip]; ok {
		return false, nil
	}
	if err := bc.c.filter.BanIP(ctx, ip); err != nil {
		return false, err
	}
	bc.blackMap[ip] = struct{}{}
	return true, nil
}
