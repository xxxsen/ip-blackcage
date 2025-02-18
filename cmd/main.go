package main

import (
	"context"
	"database/sql"
	"flag"
	ipblackcage "ip-blackcage"
	"ip-blackcage/blocker"
	"ip-blackcage/config"
	"ip-blackcage/dao"
	"ip-blackcage/ipevent"
	"ip-blackcage/route"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/glebarez/go-sqlite"
	"github.com/xxxsen/common/logger"
	"go.uber.org/zap"
)

var conf = flag.String("config", "./config.json", "config")

func main() {
	flag.Parse()
	c, err := config.Parse(*conf)
	if err != nil {
		log.Fatalf("parse config failed, err:%v", err)
	}
	log.Printf("config init succ, config:%+v", *c)
	logkit := logger.Init(c.LogConfig.File, c.LogConfig.Level, int(c.LogConfig.FileCount), int(c.LogConfig.FileSize), int(c.LogConfig.KeepDays), c.LogConfig.Console)
	//初始化ip blocker
	ipt, err := blocker.NewBlocker()
	if err != nil {
		logkit.Fatal("init blocker failed", zap.Error(err))
	}
	portlist, err := c.DecodePortList()
	if err != nil {
		logkit.Fatal("decode port list failed", zap.Error(err))
	}
	//探测当前的出口网卡
	listenInterface, err := detectValidInterface(c.Interface)
	if err != nil {
		logkit.Fatal("detect default network interface failed", zap.Error(err))
	}
	if len(c.Interface) == 0 {
		logkit.Info("detect default network interface succ", zap.String("interface", listenInterface))
	}
	//初始化ip事件读取器
	evr, err := ipevent.NewIPEventReader(
		ipevent.WithEnablePortVisit(portlist),
		ipevent.WithListenInterface(listenInterface),
	)
	if err != nil {
		logkit.Fatal("init event reader failed", zap.Error(err))
	}
	//初始化db
	db, err := sql.Open("sqlite", c.DBFile)
	if err != nil {
		logkit.Fatal("init sqlite db failed", zap.Error(err))
	}
	dao.SetIPDB(db)
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
		ipblackcage.WithSkipIPs(c.ExitIPs...),
	)
	if err != nil {
		logkit.Fatal("init cage failed", zap.Error(err))
	}
	logkit.Info("start cage...")
	if err := cage.Run(context.Background()); err != nil {
		logkit.Fatal("run cage failed", zap.Error(err))
	}
}

func detectValidInterface(netcard string) (string, error) {
	if len(netcard) > 0 {
		return netcard, nil
	}
	return route.DetectDefaultInterface()
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
