package refund

import (
	"fmt"
	"github.com/shopspring/decimal"
	"github.com/stripe/stripe-go/v74"
	"unipay/config"
	"unipay/stripe_api"
	"unipay/tables"
)

func (t *ToolRefund) doRefundStripe(list []tables.ViewRefundPaymentInfo) error {
	if !config.Cfg.Chain.Stripe.Refund {
		return nil
	}
	if len(list) == 0 {
		return nil
	}
	for i, v := range list {
		amountRefund := v.Amount
		if v.PremiumBase.Cmp(decimal.Zero) == 1 {
			amountRefund = amountRefund.Sub(v.PremiumBase.Mul(decimal.NewFromInt(100)))
		}
		if v.PremiumPercentage.Cmp(decimal.Zero) == 1 {
			amountRefund = amountRefund.Div(v.PremiumPercentage.Add(decimal.NewFromInt(1)))
		}
		//a -> a*(1+x)+0.5 -> (a*(1+x)+0.5)*0.034+0.5
		//ax>((a+ax)+0.5)*0.034
		//ax>((a+0.5)+ax)*0.034
		//ax>(a+0.5)*0.034+0.034ax
		//(a-0.034a)x>0.034a+0.5*0.034
		//0.966ax>0.034a+0.017
		//x>(0.034a+0.017)/0.966a
		//5$: x>0.03871636
		//1000$: x>0.03521429

		// a->a*(1+x)-> (a*(1+x))*0.034+0.5
		// ax>(a+ax)*0.034+0.5
		// ax>a*0.034+a*0.034*x+0.5
		// ax-a*0.034x>a*0.034+0.5
		// 0.966ax>0.034a+0.5
		// x>(0.034a+0.5)/0.966a
		// 5$: 0.67/4.83 // 0.13871636
		// 1000$: 34.5/966 // 0.03571429

		//dec34 := decimal.NewFromFloat(0.034)
		//dec50 := decimal.NewFromFloat(50)
		//amountRefund := v.Amount.Sub(v.Amount.Mul(dec34).Add(dec50))
		r, err := stripe_api.RefundPaymentIntent(v.PayHash, amountRefund.IntPart())
		if err != nil {
			return fmt.Errorf("RefundPaymentIntent err: %s", err.Error())
		}
		if r.Status == stripe.RefundStatusSucceeded {
			if err := t.DbDao.UpdateSinglePaymentToRefunded(v.PayHash, "", 0); err != nil {
				return fmt.Errorf("UpdateSinglePaymentToRefunded err: %s[%s]", err.Error(), v.PayHash)
			}
			// callback notice
			if err = t.addCallbackNotice([]tables.ViewRefundPaymentInfo{list[i]}); err != nil {
				log.Error("addCallbackNotice err:", err.Error())
			}
		}
	}
	return nil
}
