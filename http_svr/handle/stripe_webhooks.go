package handle

import (
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/webhook"
	"io/ioutil"
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

	log.Info("ApiReq:", funcName, clientIp, remoteAddr)

	const MaxBodyBytes = int64(65536)
	body := http.MaxBytesReader(ctx.Writer, ctx.Request.Body, MaxBodyBytes)
	payload, err := ioutil.ReadAll(body)
	if err != nil {
		log.Error("ioutil.ReadAll err:", err.Error())
		apiResp.ApiRespErr(http_api.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	stripeSignature := ctx.GetHeader("Stripe-Signature")
	log.Info("stripeSignature:", stripeSignature)

	endpointSecret := config.Cfg.Chain.Stripe.EndpointSecret
	event, err := webhook.ConstructEvent(payload, req.stripeSignature, endpointSecret)
	if err != nil {
		log.Error("webhook.ConstructEven err:", err.Error())
		apiResp.ApiRespErr(http_api.ApiCodeParamsInvalid, err.Error())
		ctx.JSON(http.StatusOK, apiResp)
		return
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

	var resp RespStripeWebhooks
	apiResp.ApiRespOK(resp)
	ctx.JSON(http.StatusOK, apiResp)
}
