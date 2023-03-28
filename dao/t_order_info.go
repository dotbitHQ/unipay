package dao

import (
	"github.com/dotbitHQ/unipay/tables"
	"github.com/shopspring/decimal"
)

func (d *DbDao) CreateOrder(info tables.TableOrderInfo) error {
	return d.db.Create(&info).Error
}

func (d *DbDao) GetOrderInfo(orderId, businessId string) (info tables.TableOrderInfo, err error) {
	err = d.db.Where("order_id=? AND business_id=?",
		orderId, businessId).Find(&info).Error
	return
}

func (d *DbDao) GetOrderInfoByOrderId(orderId string) (info tables.TableOrderInfo, err error) {
	err = d.db.Where("order_id=?", orderId).Find(&info).Error
	return
}

func (d *DbDao) GetOrderByAddrWithAmount(addr string, payTokenId tables.PayTokenId, amount decimal.Decimal) (order tables.TableOrderInfo, err error) {
	err = d.db.Where("pay_address=? AND pay_token_id=? AND amount=?", addr, payTokenId, amount).
		Order("id DESC").Limit(1).Find(&order).Error
	return
}
