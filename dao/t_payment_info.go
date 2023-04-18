package dao

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
	"unipay/tables"
)

func (d *DbDao) GetLatestPaymentInfo(orderId string) (info tables.TablePaymentInfo, err error) {
	err = d.db.Where("order_id=?", orderId).Order("id DESC").
		Limit(1).Find(&info).Error
	return
}

func (d *DbDao) UpdatePaymentInfoToUnRefunded(payHash string) error {
	return d.db.Model(tables.TablePaymentInfo{}).
		Where("pay_hash=? AND pay_hash_status=? AND refund_status=?",
			payHash, tables.PayHashStatusConfirm, tables.RefundStatusDefault).
		Updates(map[string]interface{}{
			"refund_status": tables.RefundStatusUnRefund,
		}).Error
}

func (d *DbDao) CreatePayment(paymentInfo tables.TablePaymentInfo) error {
	return d.db.Clauses(clause.Insert{
		Modifier: "IGNORE",
	}).Create(&paymentInfo).Error
}

func (d *DbDao) UpdatePaymentStatus(paymentInfo tables.TablePaymentInfo, noticeInfo tables.TableNoticeInfo) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(tables.TableOrderInfo{}).
			Where("order_id=? AND pay_status=?",
				paymentInfo.OrderId, tables.PayStatusUnpaid).
			Updates(map[string]interface{}{
				"pay_status": tables.PayStatusPaid,
			}).Error; err != nil {
			return err
		}

		//if err := tx.Model(tables.TablePaymentInfo{}).
		//	Where("order_id=? AND pay_hash!=? AND pay_hash_status=? AND refund_status=?",
		//		paymentInfo.OrderId, paymentInfo.PayHash, tables.PayHashStatusConfirm, tables.RefundStatusDefault).
		//	Updates(map[string]interface{}{
		//		"refund_status": tables.RefundStatusUnRefund,
		//	}).Error; err != nil {
		//	return err
		//}

		if err := tx.Clauses(clause.Insert{
			Modifier: "IGNORE",
		}).Create(&paymentInfo).Error; err != nil {
			return err
		}

		if err := tx.Clauses(clause.Insert{
			Modifier: "IGNORE",
		}).Create(&noticeInfo).Error; err != nil {
			return err
		}

		if err := tx.Model(tables.TablePaymentInfo{}).
			Where("pay_hash=? AND order_id=?",
				paymentInfo.PayHash, paymentInfo.OrderId).
			Updates(map[string]interface{}{
				"pay_address":     paymentInfo.PayAddress,
				"algorithm_id":    paymentInfo.AlgorithmId,
				"timestamp":       paymentInfo.Timestamp,
				"amount":          paymentInfo.Amount,
				"pay_hash_status": paymentInfo.PayHashStatus,
			}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (d *DbDao) GetRefundListWithin3d() (list []tables.TablePaymentInfo, err error) {
	timestamp := time.Now().Add(time.Hour * 24 * 3).Unix()
	err = d.db.Where("timestamp>=? AND pay_hash_status=? AND refund_status=?",
		timestamp, tables.PayHashStatusConfirm, tables.RefundStatusUnRefund).Find(&list).Error
	return
}

func (d *DbDao) UpdatePaymentListToRefunded(payHashList []string, refundHash string) error {
	return d.db.Model(tables.TablePaymentInfo{}).
		Where("pay_hash IN(?) AND pay_hash_status=? AND refund_status=?",
			payHashList, tables.PayHashStatusConfirm, tables.RefundStatusUnRefund).
		Updates(map[string]interface{}{
			"refund_status": tables.RefundStatusRefunded,
			"refund_hash":   refundHash,
		}).Error
}

func (d *DbDao) UpdateSinglePaymentToRefunded(payHash string, refundHash string, refundNonce uint64) error {
	return d.db.Model(tables.TablePaymentInfo{}).
		Where("pay_hash=? AND pay_hash_status=? AND refund_status=?",
			payHash, tables.PayHashStatusConfirm, tables.RefundStatusUnRefund).
		Updates(map[string]interface{}{
			"refund_status": tables.RefundStatusRefunded,
			"refund_hash":   refundHash,
			"refund_nonce":  refundNonce,
		}).Error
}

func (d *DbDao) UpdatePaymentListToUnRefunded(payHashList []string) error {
	return d.db.Model(tables.TablePaymentInfo{}).
		Where("pay_hash IN(?) AND pay_hash_status=? AND refund_status=?",
			payHashList, tables.PayHashStatusConfirm, tables.RefundStatusRefunded).
		Updates(map[string]interface{}{
			"refund_status": tables.RefundStatusUnRefund,
		}).Error
}

func (d *DbDao) UpdateSinglePaymentToUnRefunded(payHash string) error {
	return d.db.Model(tables.TablePaymentInfo{}).
		Where("pay_hash=? AND pay_hash_status=? AND refund_status=?",
			payHash, tables.PayHashStatusConfirm, tables.RefundStatusRefunded).
		Updates(map[string]interface{}{
			"refund_status": tables.RefundStatusUnRefund,
			"refund_hash":   "",
			"refund_nonce":  0,
		}).Error
}

func (d *DbDao) GetRefundNonce(refundNonce uint64, payTokenIds []tables.PayTokenId) (info tables.TablePaymentInfo, err error) {
	err = d.db.Where("refund_nonce>=? AND pay_token_id IN(?)",
		refundNonce, payTokenIds).Limit(1).Find(&info).Error
	return
}

func (d *DbDao) GetPaymentByPayHashListWithStatus(payHashList []string) (list []tables.TablePaymentInfo, err error) {
	if len(payHashList) == 0 {
		return
	}
	err = d.db.Where("pay_hash IN(?) AND pay_hash_status=? AND refund_status=?",
		payHashList, tables.PayHashStatusConfirm, tables.RefundStatusDefault).Find(&list).Error
	return
}

func (d *DbDao) GetPaymentByPayHashList(payHashList []string) (list []tables.TablePaymentInfo, err error) {
	if len(payHashList) == 0 {
		return
	}
	err = d.db.Where("pay_hash IN(?)", payHashList).Find(&list).Error
	return
}

func (d *DbDao) GetPaymentListByOrderIds(orderIds []string) (list []tables.TablePaymentInfo, err error) {
	err = d.db.Where("order_id IN(?)", orderIds).
		Order("order_id,id").Find(&list).Error
	return
}
