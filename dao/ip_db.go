package dao

import (
	"context"
	"ip-blackcage/dao/db"
)

var (
	dbInst db.IDatabase
)

func SetIPDB(dbs db.IDatabase) {
	dbInst = dbs
}

func GetIPDB(ctx context.Context) db.IDatabase {
	return dbInst
}
