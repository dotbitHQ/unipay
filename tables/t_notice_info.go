package tables

import (
	"time"
)

type TableNoticeInfo struct {
	Id           uint64       `json:"id" gorm:"column:id" gorm:"column:id; primaryKey; type:bigint(20) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '';"`
	OrderId      string       `json:"order_id" gorm:"column:order_id; uniqueIndex:uk_order_id; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	EventType    EventType    `json:"event_type" gorm:"column:event_type; type:varchar(255) NOT NULL DEFAULT '' COMMENT 'ORDER.PAY, ORDER.REFUND';"`
	NoticeCount  int          `json:"notice_count" gorm:"column:notice_count; type:smallint(6) NOT NULL DEFAULT '0' COMMENT '';"`
	NoticeStatus NoticeStatus `json:"notice_status" gorm:"column:notice_status; type:smallint(6) NOT NULL DEFAULT '0' COMMENT '0-Default 1-OK';"`
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

type EventType string

const (
	EventTypeOrderPay    EventType = "ORDER.PAY"
	EventTypeOrderRefund EventType = "ORDER.REFUND"
)

type NoticeStatus int

const (
	NoticeStatusDefault NoticeStatus = 0
	NoticeStatusOK      NoticeStatus = 1
)
