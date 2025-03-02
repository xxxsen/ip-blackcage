package dbkit

type scanRowConfig struct {
	tagname string
	limit   int
}

type ScanRowOption func(c *scanRowConfig)

func ScanWithTagName(n string) ScanRowOption {
	return func(c *scanRowConfig) {
		c.tagname = n
	}
}

func ScanDataSetLength(v int) ScanRowOption {
	return func(c *scanRowConfig) {
		c.limit = v
	}
}
