package refund

import (
	"fmt"
	"unipay/tables"
)

func (t *ToolRefund) doRefund() error {
	// get refund list
	list, err := t.DbDao.GetRefundListWithin3d()
	if err != nil {
		return fmt.Errorf("GetRefundListWithin3d err: %s", err.Error())
	}

	//
	var ckbList []tables.TablePaymentInfo
	var dogeList []tables.TablePaymentInfo
	var otherList []tables.TablePaymentInfo
	for i, v := range list {
		if v.PayHashStatus != tables.PayHashStatusConfirm && v.RefundStatus != tables.RefundStatusUnRefunded {
			continue
		}
		switch v.PayTokenId {
		case tables.PayTokenIdCKB, tables.PayTokenIdDAS:
			ckbList = append(ckbList, list[i])
		case tables.PayTokenIdETH, tables.PayTokenIdBNB, tables.PayTokenIdMATIC, tables.PayTokenIdTRX:
			otherList = append(otherList, list[i])
		case tables.PayTokenIdDOGE:
			dogeList = append(dogeList, list[i])
		default:
			log.Warn("unknown pay token id[%s]", v.PayTokenId)
		}
	}
	// todo refund

	return nil
}
