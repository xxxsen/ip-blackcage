package main

import (
	"context"
	"flag"
	ipblackcage "ip-blackcage"
	"ip-blackcage/blocker"
	"ip-blackcage/config"
	"ip-blackcage/event/ipevent"
	"log"

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
	logkit := logger.Init(c.LogConfig.File, c.LogConfig.Level, int(c.LogConfig.FileCount), int(c.LogConfig.FileSize), int(c.LogConfig.KeepDays), c.LogConfig.Console)
	ipt, err := blocker.NewIPTable(c.BlackListSet)
	if err != nil {
		logkit.Fatal("init blocker failed", zap.Error(err))
	}
	portlist, err := c.DecodePortList()
	if err != nil {
		logkit.Fatal("decode port list failed", zap.Error(err))
	}
	evr, err := ipevent.NewIPEventReader(
		ipevent.WithEnablePortVisit(portlist),
		ipevent.WithListenInterface(detectValidInterface()),
	)
	if err != nil {
		logkit.Fatal("init event reader failed", zap.Error(err))
	}
	cage, err := ipblackcage.New(
		ipblackcage.WithEventReader(evr),
		ipblackcage.WithBlocker(ipt),
	)
	if err != nil {
		logkit.Fatal("init cage failed", zap.Error(err))
	}
	logkit.Info("start cage...")
	if err := cage.Run(context.Background()); err != nil {
		logkit.Fatal("run cage failed", zap.Error(err))
	}
}

func detectValidInterface() string {
	//TODO: finish it
	return "eth0"
}
