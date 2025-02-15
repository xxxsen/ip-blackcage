package blocker

import "context"

type IBlocker interface {
	Init(ctx context.Context, ips []string) error
	BanIP(ctx context.Context, ip string) error
	UnBanIP(ctx context.Context, ip string) error
}
