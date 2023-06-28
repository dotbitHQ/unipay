package refund

import (
	"fmt"
	"unipay/config"
	"unipay/notify"
	"unipay/tables"
)

func sendRefundNotify(id uint64, payTokenId tables.PayTokenId, orderId, err string) {
	msg := fmt.Sprintf("ID: %d\nPayTokenId: %s\nOrderId: %s\nErr: %s", id, payTokenId, orderId, err)
	notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "sendRefundNotify", msg)
}
