package stripe_api

import (
	"fmt"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/paymentintent"
	"github.com/stripe/stripe-go/v74/refund"
	"unipay/config"
)

func Init() {
	stripe.Key = config.Cfg.Server.StripeKey
}

func CreatePaymentIntent(orderId string, amount int64) (*stripe.PaymentIntent, error) {
	params := &stripe.PaymentIntentParams{
		Amount:             stripe.Int64(amount),
		PaymentMethodTypes: stripe.StringSlice([]string{string(stripe.ChargePaymentMethodDetailsTypeCard)}),
		Currency:           stripe.String(string(stripe.CurrencyUSD)),
	}
	params.Metadata = map[string]string{"order_id": orderId}
	pi, err := paymentintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("paymentintent.New err: %s", err.Error())
	}
	return pi, nil
}

func GetPaymentIntent(id string) (*stripe.PaymentIntent, error) {
	pi, err := paymentintent.Get(id, nil)
	if err != nil {
		return nil, fmt.Errorf("paymentintent.Get err: %s[%s]", err.Error(), id)
	}
	return pi, nil
}

func CancelPaymentIntent(id string) (*stripe.PaymentIntent, error) {
	pi, err := paymentintent.Cancel(id, nil)
	if err != nil {
		return nil, fmt.Errorf("paymentintent.Cancel err: %s[%s]", err.Error(), id)
	}
	return pi, nil
}

func RefundPaymentIntent(id string, amount int64) (*stripe.Refund, error) {
	params := stripe.RefundParams{
		Amount:        stripe.Int64(amount),
		PaymentIntent: stripe.String(id),
	}
	r, err := refund.New(&params)
	if err != nil {
		return nil, fmt.Errorf("refund.New err: %s[%s]", err.Error(), id)
	}
	return r, nil
}
