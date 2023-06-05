package dao

import (
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"unipay/tables"
)

func (d *DbDao) CreateOrderInfoWithPaymentInfo(orderInfo tables.TableOrderInfo, paymentInfo tables.TablePaymentInfo) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&orderInfo).Error; err != nil {
			return err
		}
		if paymentInfo.PayHash != "" {
			if err := tx.Create(&paymentInfo).Error; err != nil {
				return err
			}
		}
		return nil
	})
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
	err = d.db.Where("pay_address=? AND pay_token_id=? AND amount=? AND pay_status=?", addr, payTokenId, amount, tables.PayStatusUnpaid).
		Order("id DESC").Limit(1).Find(&order).Error
	return
}
