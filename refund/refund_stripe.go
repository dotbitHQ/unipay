package refund

import (
	"fmt"
	"github.com/shopspring/decimal"
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
	for i, v := range list {
		dec34 := decimal.NewFromFloat(0.034)
		dec50 := decimal.NewFromFloat(50)

		amountRefund := v.Amount.Sub(v.Amount.Mul(dec34).Add(dec50))
		r, err := stripe_api.RefundPaymentIntent(v.PayHash, amountRefund.IntPart())
		if err != nil {
			return fmt.Errorf("RefundPaymentIntent err: %s", err.Error())
		}
		if r.Status == stripe.RefundStatusSucceeded {
			if err := t.DbDao.UpdateSinglePaymentToRefunded(v.PayHash, "", 0); err != nil {
				return fmt.Errorf("UpdateSinglePaymentToRefunded err: %s[%s]", err.Error(), v.PayHash)
			}
			// callback notice
			if err = t.addCallbackNotice([]tables.TablePaymentInfo{list[i]}); err != nil {
				log.Error("addCallbackNotice err:", err.Error())
			}
		}
	}
	return nil
}
