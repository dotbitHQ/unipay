package handle

import (
	"encoding/json"
	"fmt"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/webhook"
	"net/http"
	"unipay/config"
)

type ReqStripeWebhooks struct {
	stripe.Event
	stripeSignature string
}

type RespStripeWebhooks struct {
}

func (h *HttpHandle) StripeWebhooks(ctx *gin.Context) {
	var (
		funcName             = "StripeWebhooks"
		clientIp, remoteAddr = GetClientIp(ctx)
		req                  ReqStripeWebhooks
		apiResp              http_api.ApiResp
		err                  error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddr)
		apiResp.ApiRespErr(http_api.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	stripeSignature := ctx.GetHeader("Stripe-Signature")
	log.Info("stripeSignature:", stripeSignature)
	log.Info("ApiReq:", funcName, clientIp, remoteAddr, toolib.JsonString(req))
	req.stripeSignature = stripeSignature

	if err = h.doStripeWebhooks(&req, &apiResp); err != nil {
		log.Error("doStripeWebhooks err:", err.Error(), funcName, clientIp, remoteAddr)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doStripeWebhooks(req *ReqStripeWebhooks, apiResp *http_api.ApiResp) error {
	var resp RespStripeWebhooks

	payload, err := json.Marshal(req.Event)
	if err != nil {
		apiResp.ApiRespErr(http_api.ApiCodeParamsInvalid, err.Error())
		return fmt.Errorf("json.Marshal err: %s", err.Error())
	}

	endpointSecret := config.Cfg.Chain.Stripe.EndpointSecret
	event, err := webhook.ConstructEvent(payload, req.stripeSignature, endpointSecret)
	if err != nil {
		apiResp.ApiRespErr(http_api.ApiCodeParamsInvalid, err.Error())
		return fmt.Errorf("webhook.ConstructEvent err: %s", err.Error())
	}
	log.Info("doStripeWebhooks", event.Type, event.GetObjectValue("id", "metadata"))
	switch event.Type {
	case "charge.refunded":
	case "charge.succeeded":
	case "payment_intent.canceled":
	case "payment_intent.created":
	case "payment_intent.payment_failed":
	case "payment_intent.succeeded":
	default:
		log.Warnf("doStripeWebhooks: unknown event type [%s]", req.Event.Type)
	}

	apiResp.ApiRespOK(resp)
	return nil
}
