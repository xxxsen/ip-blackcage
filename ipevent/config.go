package ipevent

type config struct {
	iface   string
	exitIps map[string]struct{}
	portMap map[uint16]struct{}
}

type Option func(c *config)

func WithEnablePortVisit(ports []uint16) Option {
	return func(c *config) {
		for _, p := range ports {
			c.portMap[p] = struct{}{}
		}
	}
}

func WithExitIface(iface string) Option {
	return func(c *config) {
		c.iface = iface
	}
}

func WithExitIps(ips []string) Option {
	return func(c *config) {
		for _, ip := range ips {
			c.exitIps[ip] = struct{}{}
		}
	}
}

func applyOpts(opts ...Option) *config {
	c := &config{
		exitIps: make(map[string]struct{}),
		portMap: make(map[uint16]struct{}),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}
