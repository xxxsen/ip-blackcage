package ipset

type config struct {
	params []string
}

func (c *config) addParam(ps ...string) {
	c.params = append(c.params, ps...)
}

type CmdOption func(c *config)

func WithExist() CmdOption {
	return func(c *config) {
		c.addParam("-exist")
	}
}

// -exist | -output { plain | save | xml } | -quiet | -resolve | -sorted | -name |  -terse  |  -file filename
func WithOutput(typ OutputType) CmdOption {
	return func(c *config) {
		c.addParam("-output", string(typ))
	}
}

func WithQuiet() CmdOption {
	return func(c *config) {
		c.addParam("-quiet")
	}
}

func WithResolve() CmdOption {
	return func(c *config) {
		c.addParam("-resolve")
	}
}

func WithSorted() CmdOption {
	return func(c *config) {
		c.addParam("-sorted")
	}
}

func WithName() CmdOption {
	return func(c *config) {
		c.addParam("-name")
	}
}

func WithTerse() CmdOption {
	return func(c *config) {
		c.addParam("-terse")
	}
}

func WithFile(fname string) CmdOption {
	return func(c *config) {
		c.addParam("-file", fname)
	}
}

func applyOpts(opts ...CmdOption) *config {
	c := &config{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}
