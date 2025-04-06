package dao

import (
	"context"
	"fmt"
	"ip-blackcage/db"
	"ip-blackcage/model"
	"time"

	"github.com/didi/gendry/builder"
	"github.com/xxxsen/common/database"
	"github.com/xxxsen/common/database/dbkit"
)

type ListBlackIPCallback func(ctx context.Context, ips []*model.BlackCageTab) error

type IIPDBDao interface {
	AddBlackIP(ctx context.Context, ip string, remark string) error
	IncrBlackIPVisit(ctx context.Context, ip string) error
	GetBlackIP(ctx context.Context, ip string) (*model.BlackCageTab, bool, error)
	DelBlackIP(ctx context.Context, ip string) (bool, error)
	ScanBlackIP(ctx context.Context, limit int, cb ListBlackIPCallback) (int64, error)
	ListBlackIP(ctx context.Context, cond *model.ListBlackIPCondition, offset, limit int64) ([]*model.BlackCageTab, error)
}

type ipDBDaoImpl struct {
	dbc func(ctx context.Context) database.IDatabase
}

func NewIPDBDao() (IIPDBDao, error) {
	impl := &ipDBDaoImpl{
		dbc: db.GetClient,
	}
	if err := impl.init(); err != nil {
		return nil, err
	}
	return impl, nil
}

func (d *ipDBDaoImpl) getClient(ctx context.Context) database.IDatabase {
	return d.dbc(ctx)
}

func (d *ipDBDaoImpl) init() error {
	initItems := []struct {
		name string
		sql  string
	}{
		{
			name: "create table",
			sql: `
CREATE TABLE IF NOT EXISTS ip_blackcage_tab (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    remark TEXT NOT NULL,
    ctime INTEGER NOT NULL,
    mtime INTEGER NOT NULL,
    ip TEXT NOT NULL UNIQUE,
	counter INTEGER NOT NULL
);
`,
		},
		{
			name: "add_mtime_index",
			sql:  "CREATE INDEX IF NOT EXISTS idx_mtime ON ip_blackcage_tab(mtime);",
		},
	}
	for _, item := range initItems {
		if _, err := d.getClient(context.Background()).
			ExecContext(context.Background(), item.sql); err != nil {

			return fmt.Errorf("exec sql failed, job:%s, err:%w", item.name, err)
		}
	}
	return nil
}

func (d *ipDBDaoImpl) table() string {
	return "ip_blackcage_tab"
}

func (d *ipDBDaoImpl) AddBlackIP(ctx context.Context, ip string, remark string) error {
	client := d.getClient(ctx)
	now := time.Now().UnixMilli()
	sql := fmt.Sprintf(`insert or ignore into %s(remark, ctime, mtime, ip, counter) values(?, ?, ?, ?, ?)`, d.table())
	if _, err := client.ExecContext(ctx, sql, remark, now, now, ip, 1); err != nil {
		return err
	}
	return nil
}

func (d *ipDBDaoImpl) IncrBlackIPVisit(ctx context.Context, ip string) error {
	client := d.getClient(ctx)
	now := time.Now().UnixMilli()
	sql := fmt.Sprintf("update %s set counter = counter + 1, mtime = ? where ip = ?", d.table())
	if _, err := client.ExecContext(ctx, sql, now, ip); err != nil {
		return err
	}
	return nil
}

func (d *ipDBDaoImpl) GetBlackIP(ctx context.Context, ip string) (*model.BlackCageTab, bool, error) {
	where := map[string]interface{}{
		"ip":     ip,
		"_limit": []uint{0, 1},
	}
	rs := make([]*model.BlackCageTab, 0, 1)
	client := d.getClient(ctx)
	if err := dbkit.SimpleQuery(ctx, client, d.table(), where, &rs, dbkit.ScanWithTagName("json")); err != nil {
		return nil, false, err
	}
	if len(rs) == 0 {
		return nil, false, nil
	}
	return rs[0], true, nil
}

func (d *ipDBDaoImpl) DelBlackIP(ctx context.Context, ip string) (bool, error) {
	where := map[string]interface{}{
		"ip": ip,
	}
	sql, args, err := builder.BuildDelete(d.table(), where)
	if err != nil {
		return false, fmt.Errorf("build delete failed, err:%w", err)
	}
	client := d.getClient(ctx)
	rs, err := client.ExecContext(ctx, sql, args...)
	if err != nil {
		return false, err
	}
	cnt, err := rs.RowsAffected()
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (d *ipDBDaoImpl) ListBlackIP(ctx context.Context,
	cond *model.ListBlackIPCondition, offset, limit int64) ([]*model.BlackCageTab, error) {
	if cond.MtimeBetween != nil && len(cond.MtimeBetween) != 2 {
		return nil, fmt.Errorf("mtime_between should has 2 elements, get:%d", len(cond.MtimeBetween))
	}
	where := map[string]interface{}{
		"_limit": []uint{uint(offset), uint(limit)},
	}
	if cond.MtimeBetween != nil {
		where["mtime >="] = cond.MtimeBetween[0]
		where["mtime <"] = cond.MtimeBetween[1]
	}
	rs := make([]*model.BlackCageTab, 0, limit)
	if err := dbkit.SimpleQuery(ctx, d.getClient(ctx), d.table(), where, &rs, dbkit.ScanWithTagName("json")); err != nil {
		return nil, err
	}
	return rs, nil
}

func (d *ipDBDaoImpl) ScanBlackIP(ctx context.Context, limit int, cb ListBlackIPCallback) (int64, error) {
	var lastid int64 = 0
	var total int64
	for {
		rs, err := d.selectByScan(ctx, lastid, limit)
		if err != nil {
			return 0, err
		}
		if len(rs) > 0 {
			if err := cb(ctx, rs); err != nil {
				return 0, err
			}
			total += int64(len(rs))
			lastid = int64(rs[len(rs)-1].ID)
		}
		if len(rs) < limit {
			break
		}
	}
	return total, nil
}

func (d *ipDBDaoImpl) selectByScan(ctx context.Context, id int64, limit int) ([]*model.BlackCageTab, error) {
	where := map[string]interface{}{
		"id >":     id,
		"_orderby": "id asc",
		"_limit":   []uint{0, uint(limit)},
	}
	client := d.getClient(ctx)
	rs := make([]*model.BlackCageTab, 0, limit)
	if err := dbkit.SimpleQuery(ctx, client, d.table(), where, &rs, dbkit.ScanWithTagName("json")); err != nil {
		return nil, err
	}
	return rs, nil
}
