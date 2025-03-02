package ipblackcage

import (
	"context"
	"fmt"
	"ip-blackcage/event"
	"ip-blackcage/ipevent"
	"ip-blackcage/model"
	"ip-blackcage/utils"
	"time"

	"github.com/xxxsen/common/logutil"
	"go.uber.org/zap"
)

var (
	defaultIPv4LocalNetworkIPs = []string{
		"10.0.0.0/8",
		"127.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"100.64.0.0/10",
		"169.254.0.0/16",
		"192.0.2.0/24",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"192.88.99.0/24",
		"224.0.0.0/4",
	}
)

type IPBlackCage struct {
	c    *config
	done chan bool
}

func New(opts ...Option) (*IPBlackCage, error) {
	c := applyOpts(opts...)
	if c.filter == nil {
		return nil, fmt.Errorf("no filter found")
	}
	if c.obs == nil {
		return nil, fmt.Errorf("no observer found")
	}
	return &IPBlackCage{c: c, done: make(chan bool)}, nil
}

func (bc *IPBlackCage) readBlackListFromDB(ctx context.Context) ([]string, error) {
	dbIPList := make([]string, 0, 1024)
	//DB IP 列表
	delims := uint64(time.Now().Add(-1 * bc.c.banTime).UnixMilli())
	_, err := bc.c.ipDao.ScanBlackIP(ctx, 200, func(ctx context.Context, ips []*model.BlackCageTab) error {
		for _, ip := range ips {
			//仅提取满足条件的黑名单ip
			if ip.MTime <= delims {
				continue
			}
			dbIPList = append(dbIPList, ip.IP)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return dbIPList, nil
}

func (bc *IPBlackCage) readListFromFiles(lst []string) ([]string, error) {
	rs := make([]string, 0, 1024)
	for _, ipfile := range lst {
		ips, err := utils.ReadIPListFromFile(ipfile)
		if err != nil {
			return nil, fmt.Errorf("read ip list from file:%s failed, err:%w", ipfile, err)
		}
		rs = append(rs, ips...)
	}
	return rs, nil
}

func (bc *IPBlackCage) readLocalNetworkList() ([]string, error) {
	return defaultIPv4LocalNetworkIPs, nil
}

func (bc *IPBlackCage) initCageChain(ctx context.Context) error {
	dbBlackIPList, err := bc.readBlackListFromDB(ctx)
	if err != nil {
		return fmt.Errorf("read db black ips failed, err:%w", err)
	}
	userBlackIPList, err := bc.readListFromFiles(bc.c.userBlackList)
	if err != nil {
		return fmt.Errorf("read user black ips failed, err:%w", err)
	}
	userWhiteIPList, err := bc.readListFromFiles(bc.c.userWhiteList)
	if err != nil {
		return fmt.Errorf("read user white ips failed, err:%w", err)
	}
	localNetworkList, err := bc.readLocalNetworkList()
	if err != nil {
		return fmt.Errorf("read user local network list failed, err:%w", err)
	}
	if bc.c.disableLocalNetworkProtect {
		localNetworkList = nil
	}

	logutil.GetLogger(ctx).Info("read white/black ips succ",
		zap.Int("db_black_ips", len(dbBlackIPList)),
		zap.Int("user_black_ips", len(userBlackIPList)),
		zap.Int("user_white_ips", len(userWhiteIPList)),
	)
	blackList := make([]string, 0, len(dbBlackIPList)+len(userBlackIPList))
	blackList = append(blackList, dbBlackIPList...)
	blackList = append(blackList, userBlackIPList...)
	whiteList := make([]string, 0, len(userWhiteIPList)+len(localNetworkList))
	whiteList = append(whiteList, userWhiteIPList...)
	whiteList = append(whiteList, localNetworkList...)

	if err := bc.c.filter.Init(ctx, blackList, whiteList); err != nil {
		return err
	}
	return nil
}

func (bc *IPBlackCage) checkShouldBanIPByRules(_ context.Context, _ *ipevent.IPEventData) bool {
	//TODO: 在这里添加其他杂七杂八的规则
	return true
}

func (bc *IPBlackCage) Stop(ctx context.Context) error {
	logutil.GetLogger(ctx).Debug("start handle stop action")
	close(bc.done)
	<-bc.done //wait
	if err := bc.c.filter.Destroy(ctx); err != nil {
		logutil.GetLogger(ctx).Error("clean blocker rules failed", zap.Error(err))
	}
	logutil.GetLogger(ctx).Debug("handle stop action finish")
	return nil
}

func (bc *IPBlackCage) Start(ctx context.Context) error {
	if err := bc.initCageChain(ctx); err != nil {
		return err
	}
	ch, err := bc.c.obs.Open(ctx)
	if err != nil {
		return err
	}
	go bc.startHandleEvent(ctx, ch)
	return nil
}

func (bc *IPBlackCage) startHandleEvent(ctx context.Context, ch <-chan event.IEventData) {
	unBanTicker := time.NewTicker(1 * time.Minute)
	defer unBanTicker.Stop()
	for {
		select {
		case ev := <-ch:
			if err := bc.handleOneEvent(ctx, ev); err != nil {
				logutil.GetLogger(ctx).Error("handle event failed", zap.Error(err), zap.String("ev_type", ev.EventType()), zap.Int64("ts", ev.Timestamp()))
				continue
			}
		case <-unBanTicker.C:
			if err := bc.unBanExpire(ctx); err != nil {
				logutil.GetLogger(ctx).Error("do unban expire failed", zap.Error(err))
				continue
			}
		case <-bc.done:
			logutil.GetLogger(ctx).Debug("event loop exit")
			return
		}
	}
}

func (bc *IPBlackCage) unBanExpire(ctx context.Context) error {
	delims := uint64(time.Now().Add(-1 * bc.c.banTime).UnixMilli())
	ips, err := bc.c.ipDao.ListBlackIP(ctx, &model.ListBlackIPCondition{
		MtimeBetween: []uint64{0, delims},
	}, 0, 100)
	if err != nil {
		return err
	}
	for _, ip := range ips {
		logger := logutil.GetLogger(ctx).With(zap.String("ip", ip.IP),
			zap.Int64("scan_count", ip.Counter),
			zap.Int64("last_visit", int64(ip.MTime)))
		if err := bc.c.filter.UnBanIP(ctx, ip.IP); err != nil {
			logger.Error("unban ip failed", zap.Error(err))
			continue
		}
		if _, err := bc.c.ipDao.DelBlackIP(ctx, ip.IP); err != nil {
			logger.Error("remove black ip from db failed", zap.Error(err))
			continue
		}
		logger.Info("unban ip succ")
	}
	return nil
}

func (bc *IPBlackCage) handleOneEvent(ctx context.Context, ev event.IEventData) error {
	evn := ev.EventType()
	ipdata := ev.Data().(*ipevent.IPEventData)
	ts := ev.Timestamp()

	if !bc.checkShouldBanIPByRules(ctx, ipdata) {
		return nil
	}
	logger := logutil.GetLogger(ctx).With(zap.String("src", fmt.Sprintf("%s:%d", ipdata.SrcIP, ipdata.SrcPort)), zap.String("dst", fmt.Sprintf("%s:%d", ipdata.DstIP, ipdata.DstPort)))
	if bc.c.viewMode {
		logger.Debug("view mode open, skip next")
		return nil
	}

	isNew, err := bc.addToBlackList(ctx, evn, ipdata, ts)
	if err != nil {
		logger.Error("add ip to black list failed", zap.Error(err))
		return err
	}
	if !isNew {
		return nil
	}
	logger.Info("add ip to black list succ", zap.Int64("ts", ts))
	return nil
}

func (bc *IPBlackCage) addToBlackList(ctx context.Context, ev string, ipdata *ipevent.IPEventData, _ int64) (bool, error) {
	_, ok, err := bc.c.ipDao.GetBlackIP(ctx, ipdata.SrcIP)
	if err != nil {
		return false, err
	}
	if ok { // 已经存在了, 那么更新计数
		_ = bc.c.ipDao.IncrBlackIPVisit(ctx, ipdata.SrcIP)
		return false, nil
	}
	if err := bc.c.filter.BanIP(ctx, ipdata.SrcIP); err != nil {
		return false, err
	}
	bc.c.ipDao.AddBlackIP(ctx, ipdata.SrcIP, fmt.Sprintf("detect_by_event:%s|%d", ev, ipdata.DstPort))
	return true, nil
}
