package dao

import (
	"fmt"
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
	timestamp := time.Now().Add(-time.Hour * 24 * 3).UnixMilli()
	err = d.db.Where("timestamp>=? AND pay_hash_status=? AND refund_status=?",
		timestamp, tables.PayHashStatusConfirm, tables.RefundStatusUnRefund).Find(&list).Error
	return
}

func (d *DbDao) GetViewRefundListWithin3d() (list []tables.ViewRefundPaymentInfo, err error) {
	timestamp := time.Now().Add(-time.Hour * 24 * 3).UnixMilli()
	sql := fmt.Sprintf(`SELECT p.*,o.payment_address FROM %s p LEFT JOIN %s o ON o.order_id=p.order_id AND p.timestamp>=? AND pay_hash_status=? AND refund_status=?`,
		tables.TableNamePaymentInfo, tables.TableNameOrderInfo)
	err = d.db.Raw(sql, timestamp, tables.PayHashStatusConfirm, tables.RefundStatusUnRefund).Find(&list).Error
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

func (d *DbDao) UpdateSinglePaymentToRefunded(payHash, refundHash string, refundNonce uint64) error {
	return d.db.Model(tables.TablePaymentInfo{}).
		Where("pay_hash=? AND pay_hash_status=? AND refund_status=?",
			payHash, tables.PayHashStatusConfirm, tables.RefundStatusUnRefund).
		Updates(map[string]interface{}{
			"refund_status": tables.RefundStatusRefunded,
			"refund_hash":   refundHash,
			"refund_nonce":  refundNonce,
		}).Error
}

func (d *DbDao) UpdateSinglePaymentToRefunded2(payHash, refundHash, paymentAddress string, refundNonce uint64) error {
	return d.db.Model(tables.TablePaymentInfo{}).
		Where("pay_hash=? AND pay_hash_status=? AND refund_status=?",
			payHash, tables.PayHashStatusConfirm, tables.RefundStatusUnRefund).
		Updates(map[string]interface{}{
			"refund_status":   tables.RefundStatusRefunded,
			"refund_hash":     refundHash,
			"refund_nonce":    refundNonce,
			"payment_address": paymentAddress,
		}).Error
}

func (d *DbDao) UpdateRefundStatusToRejected(payHash string) error {
	return d.db.Model(tables.TablePaymentInfo{}).
		Where("pay_hash=? AND pay_hash_status=? AND refund_status=?",
			payHash, tables.PayHashStatusConfirm, tables.RefundStatusUnRefund).
		Updates(map[string]interface{}{
			"refund_status": tables.RefundStatusRefuseToRefund,
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
	err = d.db.Where("refund_nonce>=? AND payment_address=?",
		refundNonce, payTokenIds).Limit(1).Find(&info).Error
	return
}

func (d *DbDao) GetRefundNonce2(refundNonce uint64, paymentAddress string) (info tables.TablePaymentInfo, err error) {
	err = d.db.Where("refund_nonce>=? AND payment_address=?",
		refundNonce, paymentAddress).Limit(1).Find(&info).Error
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

func (d *DbDao) GetPaymentInfoByPayHash(payHash string) (info tables.TablePaymentInfo, err error) {
	err = d.db.Where("pay_hash=?", payHash).Find(&info).Error
	return
}

func (d *DbDao) GetUnPayListByTokenIdWithin3d(tokenId tables.PayTokenId) (list []tables.TablePaymentInfo, err error) {
	timestamp := tables.GetEfficientPaymentTimestamp()
	err = d.db.Where("timestamp>=? AND pay_token_id=? AND pay_hash_status=?",
		timestamp, tokenId, tables.PayHashStatusPending).Find(&list).Error
	return
}

func (d *DbDao) GetUnPayListByTokenIdMoreThan3d(tokenId tables.PayTokenId) (list []tables.TablePaymentInfo, err error) {
	timestampStart := tables.GetEfficientPaymentTimestamp()
	timestampEnd := time.Now().Add(-time.Hour * 24 * 4).UnixMilli()
	err = d.db.Where("timestamp<? AND timestamp>=? AND pay_token_id=? AND pay_hash_status=?",
		timestampStart, timestampEnd, tokenId, tables.PayHashStatusPending).Find(&list).Error
	return
}

func (d *DbDao) UpdatePayHashStatusToFailed(payHash string) error {
	return d.db.Model(tables.TablePaymentInfo{}).
		Where("pay_hash=? AND pay_hash_status=?", payHash, tables.PayHashStatusPending).
		Updates(map[string]interface{}{
			"pay_hash_status": tables.PayHashStatusFail,
		}).Error
}

func (d *DbDao) GetPaymentInfoByOrderId(orderId string) (info tables.TablePaymentInfo, err error) {
	err = d.db.Where("order_id=?", orderId).Find(&info).Limit(1).Error
	return
}
