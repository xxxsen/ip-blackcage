package ipblackcage

import (
	"ip-blackcage/blocker"
	"ip-blackcage/event"
)

type config struct {
	filter   blocker.IBlocker
	obs      event.IEventReader
	savefile string
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

func WithAutoSaveFile(f string) Option {
	return func(c *config) {
		c.savefile = f
	}
}

func applyOpts(opts ...Option) *config {
	c := &config{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}
