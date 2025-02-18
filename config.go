package ipblackcage

import (
	"ip-blackcage/blocker"
	"ip-blackcage/dao"
	"ip-blackcage/event"
)

type config struct {
	filter  blocker.IBlocker
	obs     event.IEventReader
	ipDao   dao.IIPDBDao
	skipIps map[string]struct{}

	//
	userBlackList []string
	userWhiteList []string
}

type Option func(c *config)

func WithBlocker(f blocker.IBlocker) Option {
	return func(c *config) {
		c.filter = f
	}
}

func WithEventReader(ev event.IEventReader) Option {
	return func(c *config) {
		c.obs = ev
	}
}

func WithIPDBDao(d dao.IIPDBDao) Option {
	return func(c *config) {
		c.ipDao = d
	}
}

func WithUserIPBlackList(fs []string) Option {
	return func(c *config) {
		c.userBlackList = fs
	}
}

func WithUserIPWhiteList(fs []string) Option {
	return func(c *config) {
		c.userWhiteList = fs
	}
}

func applyOpts(opts ...Option) *config {
	c := &config{
		skipIps: make(map[string]struct{}),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func WithSkipIPs(ips ...string) Option {
	return func(c *config) {
		for _, ip := range ips {
			c.skipIps[ip] = struct{}{}
		}
	}
}
