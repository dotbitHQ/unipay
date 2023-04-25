package tables

import (
	"crypto/md5"
	"fmt"
	"time"
)

type TableNoticeInfo struct {
	Id           uint64       `json:"id" gorm:"column:id; primaryKey; type:bigint(20) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '';"`
	NoticeId     string       `json:"notice_id" gorm:"column:notice_id; uniqueIndex:uk_notice_id; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	EventType    EventType    `json:"event_type" gorm:"column:event_type; type:varchar(255) NOT NULL DEFAULT '' COMMENT 'ORDER.PAY, ORDER.REFUND';"`
	PayHash      string       `json:"pay_hash" gorm:"column:pay_hash; index:k_pay_hash; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	OrderId      string       `json:"order_id" gorm:"column:order_id; index:k_order_id; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	NoticeCount  int          `json:"notice_count" gorm:"column:notice_count; type:smallint(6) NOT NULL DEFAULT '0' COMMENT '';"`
	NoticeStatus NoticeStatus `json:"notice_status" gorm:"column:notice_status; type:smallint(6) NOT NULL DEFAULT '0' COMMENT '0-Default 1-OK 2-Fail';"`
	Timestamp    int64        `json:"timestamp" gorm:"column:timestamp; index:k_timestamp; type:bigint(20) NOT NULL DEFAULT '0' COMMENT '';"`
	CreatedAt    time.Time    `json:"created_at" gorm:"column:created_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '';"`
	UpdatedAt    time.Time    `json:"updated_at" gorm:"column:updated_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '';"`
}

const (
	TableNameNoticeInfo = "t_notice_info"
)

func (t *TableNoticeInfo) TableName() string {
	return TableNameNoticeInfo
}

func (t *TableNoticeInfo) InitNoticeId() {
	noticeId := fmt.Sprintf("%s%s%d", t.EventType, t.PayHash, t.Timestamp)
	t.NoticeId = fmt.Sprintf("%x", md5.Sum([]byte(noticeId)))
}

type EventType string

const (
	EventTypeOrderPay    EventType = "ORDER.PAY"
	EventTypeOrderRefund EventType = "ORDER.REFUND"
)

type NoticeStatus int

const (
	NoticeStatusDefault NoticeStatus = 0
	NoticeStatusOK      NoticeStatus = 1
	NoticeStatusFail    NoticeStatus = 2
)
