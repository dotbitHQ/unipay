package refund

import (
	"fmt"
	"github.com/stripe/stripe-go/v74"
	"unipay/config"
	"unipay/stripe_api"
	"unipay/tables"
)

func (t *ToolRefund) doRefundStripe(list []tables.TablePaymentInfo) error {
	if !config.Cfg.Chain.Stripe.Refund {
		return nil
	}
	if len(list) == 0 {
		return nil
	}
	for _, v := range list {
		r, err := stripe_api.RefundPaymentIntent(v.PayHash, v.Amount.IntPart())
		if err != nil {
			return fmt.Errorf("RefundPaymentIntent err: %s", err.Error())
		}
		if r.Status == stripe.RefundStatusSucceeded {
			if err := t.DbDao.UpdateSinglePaymentToRefunded(v.PayHash, "", 0); err != nil {
				return fmt.Errorf("UpdateSinglePaymentToRefunded err: %s[%s]", err.Error(), v.PayHash)
			}
		}
	}
	return nil
}
