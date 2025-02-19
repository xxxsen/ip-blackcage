package utils

func StringSliceDedup(items []string) []string {
	m := make(map[string]struct{}, len(items))
	for _, item := range items {
		m[item] = struct{}{}
	}
	rs := make([]string, 0, len(m))
	for k := range m {
		rs = append(rs, k)
	}
	return rs
}
