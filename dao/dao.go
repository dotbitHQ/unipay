package dao

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/http_api"
	"gorm.io/gorm"
	"unipay/config"
	"unipay/tables"
)

type DbDao struct {
	db *gorm.DB
}

func NewGormDBNotAutoMigrate(dbMysql config.DbMysql) (*DbDao, error) {
	db, err := http_api.NewGormDB(dbMysql.Addr, dbMysql.User, dbMysql.Password, dbMysql.DbName, 100, 100)
	if err != nil {
		return nil, fmt.Errorf("toolib.NewGormDB err: %s", err.Error())
	}

	return &DbDao{db: db}, nil
}

func NewGormDB(dbMysql config.DbMysql) (*DbDao, error) {
	db, err := http_api.NewGormDB(dbMysql.Addr, dbMysql.User, dbMysql.Password, dbMysql.DbName, 100, 100)
	if err != nil {
		return nil, fmt.Errorf("toolib.NewGormDB err: %s", err.Error())
	}

	if err = db.AutoMigrate(
		&tables.TableBlockParserInfo{},
		&tables.TableOrderInfo{},
		&tables.TablePaymentInfo{},
		&tables.TableNoticeInfo{},
	); err != nil {
		return nil, err
	}

	return &DbDao{db: db}, nil
}
