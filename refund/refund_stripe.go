package refund

import (
	"fmt"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/refund"
	"unipay/config"
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
		params := stripe.RefundParams{
			Amount:        stripe.Int64(v.Amount.IntPart()),
			PaymentIntent: stripe.String(v.PayHash),
		}
		r, err := refund.New(&params)
		if err != nil {
			return fmt.Errorf("refund.New err: %s[%s]", err.Error(), v.PayHash)
		}
		if r.Status == stripe.RefundStatusSucceeded {
			if err := t.DbDao.UpdateSinglePaymentToRefunded(v.PayHash, "", 0); err != nil {
				return fmt.Errorf("UpdateSinglePaymentToRefunded err: %s[%s]", err.Error(), v.PayHash)
			}
		}
	}
	return nil
}
