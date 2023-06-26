package handle

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"unipay/config"
	"unipay/stripe_api"
	"unipay/tables"
)

type ReqOrderInfo struct {
	BusinessId string `json:"business_id"`
	OrderId    string `json:"order_id"`
}

type RespOrderInfo struct {
	OrderId         string `json:"order_id"`
	ReceiptAddr     string `json:"receipt_addr"`
	ContractAddress string `json:"contract_address"`
	ClientSecret    string `json:"client_secret"`
}

func (h *HttpHandle) OrderInfo(ctx *gin.Context) {
	var (
		funcName             = "OrderInfo"
		clientIp, remoteAddr = GetClientIp(ctx)
		req                  ReqOrderInfo
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

	if err = h.doOrderInfo(&req, &apiResp); err != nil {
		log.Error("doOrderInfo err:", err.Error(), funcName, clientIp, remoteAddr)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doOrderInfo(req *ReqOrderInfo, apiResp *http_api.ApiResp) error {
	var resp RespOrderInfo

	// check business_id
	checkBusinessIds(req.BusinessId, apiResp)
	if apiResp.ErrNo != http_api.ApiCodeSuccess {
		return nil
	}

	orderInfo, err := h.DbDao.GetOrderInfo(req.OrderId, req.BusinessId)
	if err != nil {
		apiResp.ApiRespErr(http_api.ApiCodeDbError, "Failed to get order info")
		return fmt.Errorf("GetOrderInfo err: %s", err.Error())
	}
	if orderInfo.PayTokenId == tables.PayTokenIdStripeUSD {
		paymentInfo, err := h.DbDao.GetPaymentInfoByOrderId(orderInfo.OrderId)
		if err != nil {
			apiResp.ApiRespErr(http_api.ApiCodeDbError, "Failed to get payment info")
			return fmt.Errorf("GetOrderInfo err: %s", err.Error())
		} else if paymentInfo.Id == 0 {
			apiResp.ApiRespErr(http_api.ApiCodePaymentNotExist, "No payment")
			return nil
		}
		pi, err := stripe_api.GetPaymentIntent(paymentInfo.PayHash)
		if err != nil {
			apiResp.ApiRespErr(http_api.ApiCodeError500, "Failed to get payment intent")
			return fmt.Errorf("GetPaymentIntent err: %s", err.Error())
		}
		resp.ClientSecret = pi.ClientSecret
	}

	resp.OrderId = req.OrderId
	resp.ReceiptAddr = orderInfo.ReceiptAddr
	resp.ContractAddress = orderInfo.PayTokenId.GetContractAddress(config.Cfg.Server.Net)

	apiResp.ApiRespOK(resp)
	return nil
}
