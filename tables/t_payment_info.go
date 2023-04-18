package tables

import (
	"github.com/dotbitHQ/das-lib/common"
	"github.com/shopspring/decimal"
	"time"
)

type TablePaymentInfo struct {
	Id            uint64                `json:"id" gorm:"column:id; primaryKey; type:bigint(20) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '';"`
	PayHash       string                `json:"pay_hash" gorm:"column:pay_hash; uniqueIndex:uk_pay_hash; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	OrderId       string                `json:"order_id" gorm:"column:order_id; index:k_order_id; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	PayAddress    string                `json:"pay_address" gorm:"column:pay_address; index:k_pay_address; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	AlgorithmId   common.DasAlgorithmId `json:"algorithm_id" gorm:"column:algorithm_id; type:smallint(6) NOT NULL DEFAULT '0' COMMENT '3,5-EVM 4-TRON 7-DOGE';"`
	Timestamp     int64                 `json:"timestamp" gorm:"column:timestamp; index:k_timestamp; type:bigint(20) NOT NULL DEFAULT '0' COMMENT '';"`
	Amount        decimal.Decimal       `json:"amount" gorm:"column:amount; type:decimal(60,0) NOT NULL DEFAULT '0' COMMENT '';"` // diff from order
	PayTokenId    PayTokenId            `json:"pay_token_id" gorm:"column:pay_token_id; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	PayHashStatus PayHashStatus         `json:"pay_hash_status" gorm:"column:pay_hash_status; type:smallint(6) NOT NULL DEFAULT '0' COMMENT '0-Pending 1-Confirm 2-Fail';"`
	RefundStatus  RefundStatus          `json:"refund_status" gorm:"column:refund_status; type:smallint(6) NOT NULL DEFAULT '0' COMMENT '0-Default 1-UnRefunded 2-Refunded';"`
	RefundHash    string                `json:"refund_hash" gorm:"column:refund_hash; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	RefundNonce   uint64                `json:"refund_nonce" gorm:"column:refund_nonce; index:k_refund_nonce; type:int(11) NOT NULL DEFAULT '0' COMMENT '';"`
	CreatedAt     time.Time             `json:"created_at" gorm:"column:created_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '';"`
	UpdatedAt     time.Time             `json:"updated_at" gorm:"column:updated_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '';"`
}

const (
	TableNamePaymentInfo = "t_payment_info"
)

func (t *TablePaymentInfo) TableName() string {
	return TableNamePaymentInfo
}

type PayHashStatus int

const (
	PayHashStatusPending PayHashStatus = 0
	PayHashStatusConfirm PayHashStatus = 1
	PayHashStatusFail    PayHashStatus = 2
)

type RefundStatus int

const (
	RefundStatusDefault   RefundStatus = 0
	RefundStatusUnRefund  RefundStatus = 1
	RefundStatusRefunding RefundStatus = 2
	RefundStatusRefunded  RefundStatus = 3
)
