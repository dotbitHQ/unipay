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

	log.Info("doStripeWebhooks", event.Type, toolib.JsonString(&event))
	msg := ""
	switch event.Type {
	case "charge.expired", "charge.failed", "charge.refunded",
		"charge.succeeded", "charge.updated", "charge.refund.updated":
		if event.Type == "charge.refunded" {
			var charge stripe.Charge
			if err := charge.UnmarshalJSON(event.Data.Raw); err != nil {
				e = fmt.Errorf("UnmarshalJSON err: %s", err.Error())
				return
			}
			msg = fmt.Sprintf("Event: %s\nEventID: %s\nChargeID: %s\nAmount: %.2f\nAmountRefunded: %.2f\nPaymentIntentID: %s", event.Type, event.ID, charge.ID, float64(charge.Amount)/100, float64(charge.AmountRefunded)/100, charge.PaymentIntent.ID)
		}
	case "charge.dispute.created", "charge.dispute.funds_withdrawn", "charge.dispute.updated":
		if event.Type == "charge.dispute.created" {
			var dispute stripe.Dispute
			if err := dispute.UnmarshalJSON(event.Data.Raw); err != nil {
				e = fmt.Errorf("UnmarshalJSON err: %s", err.Error())
				return
			}
			fee := int64(0)
			if len(dispute.BalanceTransactions) > 0 {
				fee = dispute.BalanceTransactions[0].Fee
			}
			msg = fmt.Sprintf("Event: %s\nEventID: %s\nDisputeID: %s\nAmount: %.2f\nFee: %.2f\nReason: %s\nPaymentIntentID: %s", event.Type, event.ID, dispute.ID, float64(dispute.Amount)/100, float64(fee)/100, dispute.Reason, dispute.PaymentIntent.ID)
			notify.SendLarkTextNotifyAtAll(config.Cfg.Notify.LarkErrorKey, "Stripe Dispute", msg)
			msg = ""
			//
			pID := dispute.PaymentIntent.ID
			paymentInfo, err := h.DbDao.GetPaymentInfoByPayHash(pID)
			if err != nil {
				e = fmt.Errorf("GetPaymentInfoByPayHash err: %s[%s]", err.Error(), pID)
				return
			} else if paymentInfo.Id == 0 {
				log.Error("doStripeWebhooks: paymentInfo.Id == 0;", pID)
				httpCode = http.StatusOK
				return
			}
			orderInfo, err := h.DbDao.GetOrderInfoByOrderId(paymentInfo.OrderId)
			if err != nil {
				e = fmt.Errorf("GetOrderInfoByOrderId err: %s[%s]", err.Error(), pID)
				return
			} else if orderInfo.Id == 0 {
				log.Error("doStripeWebhooks: orderInfo.Id == 0;", pID, paymentInfo.OrderId)
				httpCode = http.StatusOK
				return
			}
			if err := h.CN.HandlePaymentToFailByDispute(paymentInfo, orderInfo); err != nil {
				e = fmt.Errorf("HandlePayment err: %s[%s]", err.Error(), pID)
				return
			}
		}
	case "payment_intent.amount_capturable_updated", "payment_intent.requires_action", "payment_intent.canceled",
		"payment_intent.created", "payment_intent.payment_failed", "payment_intent.succeeded":
		if event.Type == "payment_intent.succeeded" {
			var pi stripe.PaymentIntent
			if err := pi.UnmarshalJSON(event.Data.Raw); err != nil {
				e = fmt.Errorf("UnmarshalJSON err: %s", err.Error())
				return
			}
			paymentInfo, err := h.DbDao.GetPaymentInfoByPayHash(pi.ID)
			if err != nil {
				e = fmt.Errorf("GetPaymentInfoByPayHash err: %s[%s]", err.Error(), pi.ID)
				return
			} else if paymentInfo.Id == 0 {
				log.Error("doStripeWebhooks: paymentInfo.Id == 0;", pi.ID)
				httpCode = http.StatusOK
				return
			}
			orderInfo, err := h.DbDao.GetOrderInfoByOrderId(paymentInfo.OrderId)
			if err != nil {
				e = fmt.Errorf("GetOrderInfoByOrderId err: %s[%s]", err.Error(), pi.ID)
				return
			} else if orderInfo.Id == 0 {
				log.Error("doStripeWebhooks: orderInfo.Id == 0;", pi.ID, paymentInfo.OrderId)
				httpCode = http.StatusOK
				return
			}
			if err := h.CN.HandlePayment(paymentInfo, orderInfo); err != nil {
				e = fmt.Errorf("HandlePayment err: %s[%s]", err.Error(), pi.ID)
				return
			}
			//
			account := pi.Metadata["account"]
			algorithmId := pi.Metadata["algorithm_id"]
			address := pi.Metadata["address"]
			action := pi.Metadata["action"]
			if account != "" && algorithmId != "" && address != "" {
				si := notify.StripeInfo{
					PID:         pi.ID,
					Account:     account,
					AlgorithmId: algorithmId,
					Address:     address,
					Action:      action,
					Amount:      pi.Amount,
				}
				notify.SendStripeNotify(config.Cfg.Notify.StripeKey, si)
			}
			//
			msg = fmt.Sprintf("Event: %s\nEventID: %s\nPaymentIntentID: %s\nAmount: %.2f", event.Type, event.ID, pi.ID, float64(pi.Amount)/100)
			if config.Cfg.Chain.Stripe.LargeAmount > 0 && pi.Amount > config.Cfg.Chain.Stripe.LargeAmount*100 {
				notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "Large Amount Order for Stripe", msg)
				msg = ""
			}
		}
	default:
		msg = fmt.Sprintf("Event: %s\nEventID: %s", event.Type, event.ID)
	}
	if msg != "" {
		notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "Stripe Webhooks", msg)
	}

	httpCode = http.StatusOK
	return
}
