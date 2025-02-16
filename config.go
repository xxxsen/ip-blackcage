package ipblackcage

import (
	"ip-blackcage/blocker"
	"ip-blackcage/dao"
	"ip-blackcage/event"
)

type config struct {
	filter blocker.IBlocker
	obs    event.IEventReader
	ipDao  dao.IIPDBDao
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

func applyOpts(opts ...Option) *config {
	c := &config{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}
