package ipevent

type config struct {
	iface string
	portm map[uint16]struct{}
}

type Option func(c *config)

func WithEnablePortVisit(ports []uint16) Option {
	return func(c *config) {
		for _, p := range ports {
			c.portm[p] = struct{}{}
		}
	}
}

func WithExitIface(iface string) Option {
	return func(c *config) {
		c.iface = iface
	}
}
