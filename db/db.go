package db

import (
	"context"

	"github.com/xxxsen/common/database"
	"github.com/xxxsen/common/database/sqlite"
)

var (
	ipdb database.IDatabase
)

func InitDB(f string) error {
	var err error
	if ipdb, err = sqlite.New(f); err != nil {
		return err
	}
	return nil
}

func GetClient(ctx context.Context) database.IDatabase {
	return ipdb
}
