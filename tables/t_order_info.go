package tables

import (
	"crypto/md5"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/shopspring/decimal"
	"time"
)

type TableOrderInfo struct {
	Id          uint64                `json:"id" gorm:"column:id; primaryKey; type:bigint(20) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '';"`
	OrderId     string                `json:"order_id" gorm:"column:order_id; uniqueIndex:uk_order_id; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	BusinessId  string                `json:"business_id" gorm:"column:business_id; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	PayAddress  string                `json:"pay_address" gorm:"column:pay_address; index:k_pay_address; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	AlgorithmId common.DasAlgorithmId `json:"algorithm_id" gorm:"column:algorithm_id; type:smallint(6) NOT NULL DEFAULT '0' COMMENT '3,5-EVM 4-TRON 7-DOGE';"`
	Amount      decimal.Decimal       `json:"amount" gorm:"column:amount; type:decimal(60,0) NOT NULL DEFAULT '0' COMMENT '';"`
	PayTokenId  PayTokenId            `json:"pay_token_id" gorm:"column:pay_token_id; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	PayStatus   PayStatus             `json:"pay_status" gorm:"column:pay_status; type:smallint(6) NOT NULL DEFAULT '0' COMMENT '0-Unpaid 1-Paid';"`
	OrderStatus OrderStatus           `json:"order_status" gorm:"column:order_status; type:smallint(6) NOT NULL DEFAULT '0' COMMENT '0-Normal 1-Cancel';"`
	Timestamp   int64                 `json:"timestamp" gorm:"column:timestamp; index:k_timestamp; type:bigint(20) NOT NULL DEFAULT '0' COMMENT '';"`
	CreatedAt   time.Time             `json:"created_at" gorm:"column:created_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '';"`
	UpdatedAt   time.Time             `json:"updated_at" gorm:"column:updated_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '';"`
}

const (
	TableNameOrderInfo = "t_order_info"
)

func (t *TableOrderInfo) TableName() string {
	return TableNameOrderInfo
}

type PayTokenId string

const (
	PayTokenIdETH       PayTokenId = "eth_eth"
	PayTokenIdErc20USDT PayTokenId = "eth_erc20_usdt"
	PayTokenIdTRX       PayTokenId = "tron_trx"
	PayTokenIdTrc20USDT PayTokenId = "tron_trc20_usdt"
	PayTokenIdBNB       PayTokenId = "bsc_bnb"
	PayTokenIdBep20USDT PayTokenId = "bsc_bep20_usdt"
	PayTokenIdMATIC     PayTokenId = "polygon_matic"
	PayTokenIdDOGE      PayTokenId = "doge_doge"
	PayTokenIdDAS       PayTokenId = "ckb_das"
	PayTokenIdCKB       PayTokenId = "ckb_ckb"
	PayTokenIdInternal  PayTokenId = "ckb_internal"
	PayTokenIdCoupon    PayTokenId = "coupon"
)

func (p PayTokenId) GetContractAddress(net common.DasNetType) string {
	contract := ""
	if net == common.DasNetTypeMainNet {
		switch p {
		case PayTokenIdErc20USDT:
			contract = "0xdAC17F958D2ee523a2206206994597C13D831ec7"
		case PayTokenIdBep20USDT:
			contract = "0x55d398326f99059fF775485246999027B3197955"
		case PayTokenIdTrc20USDT:
			contract = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
		}
	} else {
		switch p {
		case PayTokenIdErc20USDT:
			contract = "0xDf954C7D93E300183836CdaA01a07a1743F183EC"
		case PayTokenIdBep20USDT:
			contract = "0x5Efb0D565898be6748920db2c3BdC22BDFd5c187"
		case PayTokenIdTrc20USDT:
			contract = "TKMVcZtc1kyb2qFruhgd91mRCPNhPRRrsw"
		}
	}
	return contract
}

type PayStatus int

const (
	PayStatusUnpaid PayStatus = 0
	PayStatusPaid   PayStatus = 1
)

type OrderStatus int

const (
	OrderStatusNormal OrderStatus = 0
	OrderStatusCancel OrderStatus = 1
)

func (t *TableOrderInfo) InitOrderId() {
	orderId := fmt.Sprintf("%s%s%s%s%d", t.BusinessId, t.PayAddress, t.PayTokenId, t.Amount.String(), t.Timestamp)
	t.OrderId = fmt.Sprintf("%x", md5.Sum([]byte(orderId)))
}
