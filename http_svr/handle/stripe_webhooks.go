package handle

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/webhook"
	"io/ioutil"
	"net/http"
	"unipay/config"
	"unipay/notify"
)

func (h *HttpHandle) StripeWebhooks(ctx *gin.Context) {
	var (
		funcName             = "StripeWebhooks"
		clientIp, remoteAddr = GetClientIp(ctx)
		apiResp              http_api.ApiResp
	)

	log.Info("ApiReq:", funcName, clientIp, remoteAddr)
	httpCode, err := h.doStripeWebhooks(ctx)
	if err != nil {
		log.Error("doStripeWebhooks err: ", err.Error())
		apiResp.ApiRespErr(http_api.ApiCodeError500, err.Error())
	}

	ctx.JSON(httpCode, apiResp)
}

func (h *HttpHandle) doStripeWebhooks(ctx *gin.Context) (httpCode int, e error) {
	httpCode = http.StatusInternalServerError

	const MaxBodyBytes = int64(65536)
	body := http.MaxBytesReader(ctx.Writer, ctx.Request.Body, MaxBodyBytes)
	payload, err := ioutil.ReadAll(body)
	if err != nil {
		e = fmt.Errorf("ioutil.ReadAll err: %s", err.Error())
		return
	}
	stripeSignature := ctx.GetHeader("Stripe-Signature")
	log.Info("stripeSignature:", stripeSignature)
	//log.Info("payload:", string(payload))

	endpointSecret := config.Cfg.Chain.Stripe.EndpointSecret
	event, err := webhook.ConstructEvent(payload, stripeSignature, endpointSecret)
	if err != nil {
		e = fmt.Errorf("webhook.ConstructEven err: %s", err.Error())
		return
	}

	log.Info("doStripeWebhooks", event.Type)

	switch event.Type {
	case "charge.refunded", "charge.succeeded":
		// todo
	case "payment_intent.succeeded":
		var pi stripe.PaymentIntent
		if err := pi.UnmarshalJSON(event.Data.Raw); err != nil {
			e = fmt.Errorf("UnmarshalJSON err: %s", err.Error())
			return
		}
		log.Info("pi:", toolib.JsonString(&pi))
		if event.Type == "payment_intent.succeeded" {
			paymentInfo, err := h.DbDao.GetPaymentInfoByPayHash(pi.ID)
			if err != nil {
				e = fmt.Errorf("GetPaymentInfoByPayHash err: %s[%s]", err.Error(), pi.ID)
				return
			}
			orderInfo, err := h.DbDao.GetOrderInfoByOrderId(paymentInfo.OrderId)
			if err != nil {
				e = fmt.Errorf("GetOrderInfoByOrderId err: %s[%s]", err.Error(), pi.ID)
				return
			}
			if err := h.CN.HandlePayment(paymentInfo, orderInfo); err != nil {
				e = fmt.Errorf("HandlePayment err: %s[%s]", err.Error(), pi.ID)
				return
			}
		} else {
			msg := fmt.Sprintf("Event: %s\nID: %s", event.Type, pi.ID)
			notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "Stripe Webhooks", msg)
		}
	default:
		log.Warnf("doStripeWebhooks: unknown event type [%s] [%s]", event.Type, toolib.JsonString(&event))
	}

	httpCode = http.StatusOK
	return
}
