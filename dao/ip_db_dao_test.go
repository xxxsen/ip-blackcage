package dao

import (
	"context"
	"ip-blackcage/db"
	"ip-blackcage/model"
	"os"
	"testing"

	_ "github.com/glebarez/go-sqlite"
	"github.com/stretchr/testify/assert"
)

func TestIPDBDao(t *testing.T) {
	path := "/tmp/ip_db_test.db"
	defer os.Remove(path)
	db.InitDB(path)

	d, err := NewIPDBDao()
	assert.NoError(t, err)
	ctx := context.Background()
	{ //插入数据
		ips := []string{"1.2.3.4", "2.3.4.5", "3.4.5.6"} //duplicate
		for _, ip := range ips {
			err := d.AddBlackIP(ctx, ip, "test")
			assert.NoError(t, err)
			err = d.IncrBlackIPVisit(ctx, ip)
			assert.NoError(t, err)
		}
	}
	{ //读取全列表
		limit := 1
		cnt, err := d.ScanBlackIP(ctx, limit, func(ctx context.Context, ips []*model.BlackCageTab) error {
			for _, ip := range ips {
				t.Logf("recv ip item:%v", *ip)
				assert.Equal(t, int64(2), ip.Counter)
			}
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 3, int(cnt))
	}
	{ //获取单个ip信息
		info, ok, err := d.GetBlackIP(ctx, "1.2.3.4")
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "1.2.3.4", info.IP)
	}
	{ //删除再获取
		ok, err := d.DelBlackIP(ctx, "1.2.3.4")
		assert.NoError(t, err)
		assert.True(t, ok)
		_, ok, err = d.GetBlackIP(ctx, "1.2.3.4")
		assert.NoError(t, err)
		assert.False(t, ok)
	}
}
