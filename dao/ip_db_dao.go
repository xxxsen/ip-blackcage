package dao

import (
	"context"
	"database/sql"
	"fmt"
	"ip-blackcage/dao/db"
	"ip-blackcage/model"
)

type ListBlackIPCallback func(ctx context.Context, ips []*model.BlackCageTab) error

type IIPDBDao interface {
	AddBlackIP(ctx context.Context, bip *model.BlackCageTab) error
	GetBlackIP(ctx context.Context, ip string) (*model.BlackCageTab, bool, error)
	DelBlackIP(ctx context.Context, ip string) (bool, error)
	ListBlackIP(ctx context.Context, limit int, cb ListBlackIPCallback) (int64, error)
}

type ipDBDaoImpl struct {
	dbc func(ctx context.Context) db.IDatabase
}

func NewIPDBDao() (IIPDBDao, error) {
	impl := &ipDBDaoImpl{
		dbc: GetIPDB,
	}
	if err := impl.init(); err != nil {
		return nil, err
	}
	return impl, nil
}

func (d *ipDBDaoImpl) getClient(ctx context.Context) db.IDatabase {
	return d.dbc(ctx)
}

func (d *ipDBDaoImpl) init() error {
	sql := `
CREATE TABLE IF NOT EXISTS ip_blackcage_tab (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,
    ctime INTEGER NOT NULL,
    mtime INTEGER NOT NULL,
    ip TEXT NOT NULL UNIQUE
);
`
	_, err := d.getClient(context.Background()).ExecContext(context.Background(), sql)
	if err != nil {
		return err
	}
	return nil
}

func (d *ipDBDaoImpl) table() string {
	return "ip_blackcage_tab"
}

func (d *ipDBDaoImpl) AddBlackIP(ctx context.Context, bip *model.BlackCageTab) error {
	client := d.getClient(ctx)

	sql := fmt.Sprintf(`insert or ignore into %s(event_type, ctime, mtime, ip) values(?, ?, ?, ?)`, d.table())
	if _, err := client.ExecContext(ctx, sql, bip.EventType, bip.CTime, bip.MTime, bip.IP); err != nil {
		return err
	}
	return nil
}

func (d *ipDBDaoImpl) GetBlackIP(ctx context.Context, ip string) (*model.BlackCageTab, bool, error) {
	sql := fmt.Sprintf("select id, event_type, ctime, mtime, ip from %s where ip = ? limit 1", d.table())
	client := d.getClient(ctx)
	rows, err := client.QueryContext(ctx, sql, ip)
	if err != nil {
		return nil, false, err
	}
	res, err := d.scanIPBlackCageRows(rows)
	if err != nil {
		return nil, false, err
	}
	if len(res) == 0 {
		return nil, false, nil
	}
	return res[0], true, nil
}

func (d *ipDBDaoImpl) DelBlackIP(ctx context.Context, ip string) (bool, error) {
	sql := fmt.Sprintf("delete from %s where ip = ?", d.table())
	client := d.getClient(ctx)
	rs, err := client.ExecContext(ctx, sql, ip)
	if err != nil {
		return false, err
	}
	cnt, err := rs.RowsAffected()
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (d *ipDBDaoImpl) ListBlackIP(ctx context.Context, limit int, cb ListBlackIPCallback) (int64, error) {
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
	sql := fmt.Sprintf(`select id, event_type, ctime, mtime, ip from %s where id > ? order by id asc limit %d`, d.table(), limit)
	client := d.getClient(ctx)
	rows, err := client.QueryContext(ctx, sql, id)
	if err != nil {
		return nil, err
	}
	return d.scanIPBlackCageRows(rows)
}

func (d *ipDBDaoImpl) scanIPBlackCageRows(rows *sql.Rows) ([]*model.BlackCageTab, error) {
	defer rows.Close()
	rs := make([]*model.BlackCageTab, 0, 1)
	for rows.Next() {
		tab := &model.BlackCageTab{}
		if err := rows.Scan(&tab.ID, &tab.EventType, &tab.CTime, &tab.MTime, &tab.IP); err != nil {
			return nil, err
		}
		rs = append(rs, tab)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return rs, nil
}
