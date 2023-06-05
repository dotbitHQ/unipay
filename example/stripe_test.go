package example

import (
	"fmt"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/balance"
	"github.com/stripe/stripe-go/v74/customer"
	"github.com/stripe/stripe-go/v74/paymentintent"
	"testing"
)

const (
	stripeKey = ""
)

// https://github.com/stripe/stripe-go
func TestStripe(t *testing.T) {

}

func TestAmount(t *testing.T) {
	usdAmount, _ := decimal.NewFromString("0.4923")
	amount := usdAmount.Mul(decimal.New(1, 2)).Div(decimal.NewFromInt(1)).Ceil()
	fmt.Println(amount)
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
		Amount: stripe.Int64(100),
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
