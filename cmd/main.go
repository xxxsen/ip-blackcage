package main

import (
	"context"
	"flag"
	ipblackcage "ip-blackcage"
	"ip-blackcage/blocker"
	"ip-blackcage/config"
	"ip-blackcage/dao"
	"ip-blackcage/db"
	"ip-blackcage/ipevent"
	"ip-blackcage/route"
	"ip-blackcage/utils"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "github.com/glebarez/go-sqlite"
	"github.com/xxxsen/common/logger"
	"github.com/xxxsen/common/logutil"
	"go.uber.org/zap"
)

var conf = flag.String("config", "./config.json", "config")

func main() {
	flag.Parse()
	c, err := config.Parse(*conf)
	if err != nil {
		log.Fatalf("parse config failed, err:%v", err)
	}
	logkit := logger.Init(c.LogConfig.File, c.LogConfig.Level, int(c.LogConfig.FileCount), int(c.LogConfig.FileSize), int(c.LogConfig.KeepDays), c.LogConfig.Console)
	logkit.Info("config init succ", zap.Any("config", c))
	//初始化ip blocker
	ipt, err := blocker.NewBlocker(
		blocker.WithCageSize(c.CageSize),
	)
	if err != nil {
		logkit.Fatal("init blocker failed", zap.Error(err))
	}
	portlist, err := c.DecodePortList()
	if err != nil {
		logkit.Fatal("decode port list failed", zap.Error(err))
	}
	//重建当前的出口网卡/ip
	if err := rebuildExitIfaceName(&c.NetConfig); err != nil {
		logkit.Fatal("rebuild exit iface name failed", zap.Error(err))
	}
	if err := rebuildExitIPs(&c.NetConfig); err != nil {
		logkit.Fatal("rebuild exit ips failed", zap.Error(err))
	}
	logkit.Info("use exit iface name", zap.String("name", c.NetConfig.Interface))
	logkit.Info("use exit ips", zap.Strings("ips", c.NetConfig.ExitIPs))
	//初始化ip事件读取器
	evr, err := ipevent.NewIPEventReader(
		ipevent.WithEnablePortVisit(portlist),
		ipevent.WithExitIface(c.NetConfig.Interface),
		ipevent.WithExitIps(c.NetConfig.ExitIPs),
	)
	if err != nil {
		logkit.Fatal("init event reader failed", zap.Error(err))
	}
	//初始化db
	if err := initDB(c.DBFile); err != nil {
		logkit.Fatal("init db failed", zap.Error(err))
	}
	//初始化ip db dao
	ipdao, err := dao.NewIPDBDao()
	if err != nil {
		logkit.Fatal("init ip db dao failed", zap.Error(err))
	}
	ublist, err := resolveUserFile(c.UserIPBlackListDir, "blacklist-")
	if err != nil {
		logkit.Fatal("init user black list failed", zap.Error(err))
	}
	uwlist, err := resolveUserFile(c.UserIPWhiteListDir, "whitelist-")
	if err != nil {
		logkit.Fatal("init user white list failed", zap.Error(err))
	}
	cage, err := ipblackcage.New(
		ipblackcage.WithEventReader(evr),
		ipblackcage.WithBlocker(ipt),
		ipblackcage.WithIPDBDao(ipdao),
		ipblackcage.WithUserIPBlackList(ublist),
		ipblackcage.WithUserIPWhiteList(uwlist),
		ipblackcage.WithViewMode(c.ViewMode),
		ipblackcage.WithBanTime(time.Duration(c.BanTime)*time.Second),
		ipblackcage.WithDisableLocalNetworkProtect(c.DisableLocalNetworkProtect),
	)
	if err != nil {
		logkit.Fatal("init cage failed", zap.Error(err))
	}
	logkit.Info("start cage...")
	ctx := context.Background()
	if err := cage.Start(ctx); err != nil {
		logkit.Fatal("run cage failed", zap.Error(err))
	}
	waitSignalAndExit(ctx, cage)
}

func rebuildExitIfaceName(netc *config.NetConfig) error {
	if len(netc.Interface) > 0 {
		return nil
	}
	iface, err := route.DetectExitInterface()
	if err != nil {
		return err
	}
	netc.Interface = iface
	return nil
}

func rebuildExitIPs(netc *config.NetConfig) error {
	ips, err := route.ReadExitIP(netc.Interface)
	if err != nil {
		return err
	}
	netc.ExitIPs = utils.StringSliceDedup(append(ips, netc.ExitIPs...))
	return nil
}

func resolveUserFile(dir string, prefix string) ([]string, error) {
	if len(dir) == 0 {
		return nil, nil
	}
	rs := make([]string, 0, 32)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		name := filepath.Base(ent.Name())
		if strings.HasPrefix(name, prefix) {
			rs = append(rs, filepath.Join(dir, ent.Name()))
		}
	}
	return rs, nil
}

func initDB(f string) error {
	return db.InitDB(f)
}

func waitSignalAndExit(ctx context.Context, cage *ipblackcage.IPBlackCage) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	logutil.GetLogger(ctx).Info("recv stop signal, stop ip cage", zap.Any("signal", sig.String()))
	if err := cage.Stop(ctx); err != nil {
		logutil.GetLogger(ctx).Error("stop cage failed", zap.Error(err))
		os.Exit(1)
		return
	}
	os.Exit(0)
}
