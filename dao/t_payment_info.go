package dao

import (
	"github.com/dotbitHQ/unipay/tables"
	"github.com/shopspring/decimal"
)

func (d *DbDao) GetUnRefundedPaymentInfo(orderId string, amount decimal.Decimal) (info tables.TablePaymentInfo, err error) {
	err = d.db.Where("order_id=? AND amount=? AND pay_hash_status=? AND refund_status=?",
		orderId, amount, tables.PayHashStatusConfirm, tables.RefundStatusDefault).
		Limit(1).Find(&info).Error
	return
}

func (d *DbDao) UpdatePaymentInfoToUnRefunded(payHash string) error {
	return d.db.Model(tables.TablePaymentInfo{}).
		Where("pay_hash=? AND pay_hash_status=? AND refund_status=?",
			payHash, tables.PayHashStatusConfirm, tables.RefundStatusDefault).
		Updates(map[string]interface{}{
			"refund_status": tables.RefundStatusUnRefunded,
		}).Error
}
