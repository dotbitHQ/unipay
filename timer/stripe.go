package timer

import (
	"fmt"
	"github.com/stripe/stripe-go/v74"
	"time"
	"unipay/notify"
	"unipay/stripe_api"
	"unipay/tables"
)

func (t *ToolTimer) RunCheckStripeStatus() {
	tickerStripe := time.NewTicker(time.Second * 15)
	t.Wg.Add(1)
	go func() {
		for {
			select {
			case <-tickerStripe.C:
				if err := t.checkStripeStatus(); err != nil {
					log.Error("checkStripeStatus err: ", err.Error())
					notify.SendLarkErrNotify("checkStripeStatus", err.Error())
				}
			case <-t.Ctx.Done():
				log.Warn("RunCheckStripeStatus done")
				t.Wg.Done()
				return
			}
		}
	}()
}

func (t *ToolTimer) checkStripeStatus() error {
	//list, err := t.DbDao.GetUnPayListByTokenIdWithin3d(tables.PayTokenIdStripeUSD)
	//if err != nil {
	//	return fmt.Errorf("GetUnPayListByTokenIdWithin3d err: %s", err.Error())
	//}
	//for i, v := range list {
	//	pi, err := stripe_api.GetPaymentIntent(v.PayHash)
	//	if err != nil {
	//		return fmt.Errorf("GetPaymentIntent err: %s", err.Error())
	//	}
	//	if pi.Status == stripe.PaymentIntentStatusSucceeded {
	//		orderInfo, err := t.DbDao.GetOrderInfoByOrderId(v.OrderId)
	//		if err != nil {
	//			return fmt.Errorf("GetOrderInfoByOrderId err: %s", err.Error())
	//		}
	//		if err = t.CN.HandlePayment(list[i], orderInfo); err != nil {
	//			return fmt.Errorf("HandlePayment err: %s", err.Error())
	//		}
	//	}
	//}

	list, err := t.DbDao.GetUnPayListByTokenIdMoreThan3d(tables.PayTokenIdStripeUSD)
	if err != nil {
		return fmt.Errorf("GetUnPayListByTokenIdMoreThan3d err: %s", err.Error())
	}
	for _, v := range list {
		pi, err := stripe_api.CancelPaymentIntent(v.PayHash)
		if err != nil {
			return fmt.Errorf("CancelPaymentIntent err: %s", err.Error())
		}
		if pi.Status == stripe.PaymentIntentStatusCanceled {
			if err := t.DbDao.UpdatePayHashStatusToFailed(v.PayHash); err != nil {
				return fmt.Errorf("UpdatePayHashStatusToFailed err: %s[%s]", err.Error(), v.PayHash)
			}
		}
	}
	return nil
}
