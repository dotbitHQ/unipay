package dao

import "github.com/dotbitHQ/unipay/tables"

func (d *DbDao) CreateOrder(info tables.TableOrderInfo) error {
	return d.db.Create(&info).Error
}

func (d *DbDao) GetOrderInfo(orderId, businessId string) (info tables.TableOrderInfo, err error) {
	err = d.db.Where("order_id=? AND business_id=?",
		orderId, businessId).Find(&info).Error
	return
}
