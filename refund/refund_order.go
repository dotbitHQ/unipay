package refund

import (
	"fmt"
	"unipay/config"
	"unipay/notify"
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
	var stripeList []tables.TablePaymentInfo
	for i, v := range list {
		if v.PayHashStatus != tables.PayHashStatusConfirm && v.RefundStatus != tables.RefundStatusUnRefund {
			continue
		}
		switch v.PayTokenId {
		case tables.PayTokenIdCKB, tables.PayTokenIdDAS:
			ckbList = append(ckbList, list[i])
		case tables.PayTokenIdETH, tables.PayTokenIdBNB,
			tables.PayTokenIdMATIC, tables.PayTokenIdTRX, tables.PayTokenIdTrc20USDT,
			tables.PayTokenIdErc20USDT, tables.PayTokenIdBep20USDT:
			otherList = append(otherList, list[i])
		case tables.PayTokenIdStripeUSD:
			stripeList = append(stripeList, list[i])
		case tables.PayTokenIdDOGE:
			dogeList = append(dogeList, list[i])
		default:
			log.Warn("unknown pay token id[%s]", v.PayTokenId)
		}
	}

	// refund
	if err = t.doRefundCkb(ckbList); err != nil {
		log.Error("doRefundCkb err: ", err.Error())
		notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doRefundCKB", err.Error())
	}
	if err = t.doRefundDoge(dogeList); err != nil {
		log.Error("doRefundDoge err: ", err.Error())
		notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doRefundDoge", err.Error())
	}
	if err = t.doRefundOther(otherList); err != nil {
		log.Error("doRefundOther err: ", err.Error())
		notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doRefundOther", err.Error())
	}
	if err = t.doRefundStripe(stripeList); err != nil {
		log.Error("doRefundStripe err: ", err.Error())
		notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doRefundStripe", err.Error())
	}

	return nil
}
