package dao

import "github.com/dotbitHQ/unipay/tables"

func (d *DbDao) CreateOrder(info tables.TableOrderInfo) error {
	return d.db.Create(&info).Error
}
