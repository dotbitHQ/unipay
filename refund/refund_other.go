package refund

import (
	"fmt"
	"unipay/notify"
	"unipay/tables"
)

func sendRefundNotify(id uint64, payTokenId tables.PayTokenId, orderId, err string) {
	msg := fmt.Sprintf("ID: %d\nPayTokenId: %s\nOrderId: %s\nErr: %s", id, payTokenId, orderId, err)
	notify.SendLarkErrNotify("sendRefundNotify", msg)
}
