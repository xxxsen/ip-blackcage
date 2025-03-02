package blocker

type config struct {
	cageSize uint64
}

type Option func(c *config)

func WithCageSize(sz uint64) Option {
	return func(c *config) {
		c.cageSize = sz
	}
}

func applyOpts(opts ...Option) *config {
	c := &config{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}
