package handle

import (
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"github.com/stripe/stripe-go/v74"
	"net/http"
)

type ReqStripeWebhooks struct {
	stripe.Event
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
	log.Info("ApiReq:", funcName, clientIp, remoteAddr, toolib.JsonString(req))

	if err = h.doStripeWebhooks(&req, &apiResp); err != nil {
		log.Error("doStripeWebhooks err:", err.Error(), funcName, clientIp, remoteAddr)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doStripeWebhooks(req *ReqStripeWebhooks, apiResp *http_api.ApiResp) error {
	var resp RespStripeWebhooks

	//endpointSecret := config.Cfg.Server.StripeEndpointSecret
	//event, err := webhook.ConstructEvent(payload, req.Header.Get("Stripe-Signature"), endpointSecret)
	//if err != nil {
	//	apiResp.ApiRespErr(http_api.ApiCodeParamsInvalid, err.Error())
	//	return fmt.Errorf("webhook.ConstructEvent err: %s", err.Error())
	//}

	switch req.Event.Type {
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
