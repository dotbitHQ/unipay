package example

import (
	"fmt"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/balance"
	"github.com/stripe/stripe-go/v74/customer"
	"github.com/stripe/stripe-go/v74/paymentintent"
	"github.com/stripe/stripe-go/v74/refund"
	"testing"
)

const (
	stripeKey = ""
)

// https://github.com/stripe/stripe-go
func TestStripe(t *testing.T) {

}

func TestWebhooks(t *testing.T) {
	//fmt.Printf("%.2f", float64(500)/100)

	dec34 := decimal.NewFromFloat(0.034)
	dec50 := decimal.NewFromFloat(50)
	dec500 := decimal.NewFromInt(500)
	fmt.Println(dec500.Sub(dec500.Mul(dec34).Add(dec50)))

}

func TestRefund(t *testing.T) {
	stripe.Key = stripeKey
	params := stripe.RefundParams{
		Amount:        stripe.Int64(100),
		PaymentIntent: stripe.String(""),
	}
	r, _ := refund.New(&params)
	fmt.Println(toolib.JsonString(r))
}

func TestAmount(t *testing.T) {
	usdAmount, _ := decimal.NewFromString("0.4923")
	amount := usdAmount.Mul(decimal.New(1, 2)).Div(decimal.NewFromInt(1)).Ceil()
	fmt.Println(amount)
}

func TestConfirmPaymentIntent(t *testing.T) {
	stripe.Key = stripeKey
	params := &stripe.PaymentIntentConfirmParams{
		PaymentMethod: stripe.String("pm_card_visa"),
	}
	pi, _ := paymentintent.Confirm("", params)
	fmt.Println(toolib.JsonString(pi))
	// canceled
}

func TestCancelPaymentIntent(t *testing.T) {
	stripe.Key = stripeKey
	pi, _ := paymentintent.Cancel("", nil)
	fmt.Println(toolib.JsonString(pi))
	// canceled
}

func TestGetPaymentIntent(t *testing.T) {
	stripe.Key = stripeKey
	pi, _ := paymentintent.Get("", nil)
	fmt.Println(toolib.JsonString(pi))
	fmt.Println(pi.ClientSecret)
	// canceled
	// https://checkout.stripe.com/c/pay/cs_test_...
}

func TestCreatePaymentIntent(t *testing.T) {
	stripe.Key = stripeKey
	params := &stripe.PaymentIntentParams{
		Amount: stripe.Int64(500),
		//PaymentMethodTypes: stripe.StringSlice([]string{string(stripe.ChargePaymentMethodDetailsTypeCard)}),
		Currency: stripe.String(string(stripe.CurrencyUSD)),
	}
	params.Metadata = map[string]string{"order_id": "test0001"}
	pi, _ := paymentintent.New(params)
	fmt.Println(toolib.JsonString(&pi))
	// requires_payment_method
}

func TestBalance(t *testing.T) {
	stripe.Key = stripeKey
	b, _ := balance.Get(nil)
	fmt.Println(toolib.JsonString(&b))
}

func TestCreateCustomer(t *testing.T) {
	stripe.Key = stripeKey

	params := &stripe.CustomerParams{
		Description: stripe.String("My First Test Customer (created for API docs at https://www.stripe.com/docs/api)"),
	}
	c, _ := customer.New(params)
	fmt.Println(c.ID)
}

func TestListCustom(t *testing.T) {
	stripe.Key = stripeKey

	params := &stripe.CustomerListParams{}
	params.Filters.AddFilter("limit", "", "3")
	i := customer.List(params)
	for i.Next() {
		c := i.Customer()
		fmt.Println(c.ID)
	}
}

func TestDispute(t *testing.T) {
	var dispute stripe.Dispute
	str := ``

	if err := dispute.UnmarshalJSON([]byte(str)); err != nil {
		t.Fatal(err)
	}
	fmt.Println(dispute.ID, dispute.Amount, dispute.Charge.ID, dispute.PaymentIntent.ID)
}
