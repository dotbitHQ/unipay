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

func (d *DbDao) CreateOrderInfNoNeedPay(orderInfo tables.TableOrderInfo, paymentInfo tables.TablePaymentInfo, notice tables.TableNoticeInfo) error {
	orderInfo.PayStatus = tables.PayStatusPaid
	paymentInfo.PayHashStatus = tables.PayHashStatusConfirm
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&orderInfo).Error; err != nil {
			return err
		}
		if err := tx.Create(&paymentInfo).Error; err != nil {
			return err
		}
		if err := tx.Create(&notice).Error; err != nil {
			return err
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

func (d *DbDao) GetOrderInfoByOrderIdWithAddr(orderId, receiptAddr string) (info tables.TableOrderInfo, err error) {
	err = d.db.Where("order_id=? AND payment_address=?", orderId, receiptAddr).Find(&info).Error
	return
}

func (d *DbDao) GetOrderByAddrWithAmount(addr string, payTokenId tables.PayTokenId, amount decimal.Decimal) (order tables.TableOrderInfo, err error) {
	err = d.db.Where("pay_address=? AND pay_token_id=? AND amount=? AND pay_status=?", addr, payTokenId, amount, tables.PayStatusUnpaid).
		Order("id DESC").Limit(1).Find(&order).Error
	return
}

func (d *DbDao) GetOrderByAddrWithAmountAndAddr(addr, receiptAddr string, payTokenId tables.PayTokenId, amount decimal.Decimal) (order tables.TableOrderInfo, err error) {
	err = d.db.Where("pay_address=? AND payment_address=? AND pay_token_id=? AND amount=? AND pay_status=?",
		addr, receiptAddr, payTokenId, amount, tables.PayStatusUnpaid).
		Order("id DESC").Limit(1).Find(&order).Error
	return
}

func (d *DbDao) GetLatestOrderByAddrWithAmountAndAddr(addr, receiptAddr string, payTokenId tables.PayTokenId, amount decimal.Decimal) (order tables.TableOrderInfo, err error) {
	err = d.db.Where("pay_address=? AND payment_address=? AND pay_token_id=? AND amount=?",
		addr, receiptAddr, payTokenId, amount).
		Order("id DESC").Limit(1).Find(&order).Error
	return
}
