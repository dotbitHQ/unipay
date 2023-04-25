package handle

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"unipay/config"
)

type ReqOrderInfo struct {
	BusinessId string `json:"business_id"`
	OrderId    string `json:"order_id"`
}

type RespOrderInfo struct {
	OrderId         string `json:"order_id"`
	PaymentAddress  string `json:"payment_address"`
	ContractAddress string `json:"contract_address"`
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
		apiResp.ApiRespErr(http_api.ApiCodeDbError, "failed to get order info")
		return fmt.Errorf("GetOrderInfo err: %s", err.Error())
	}
	paymentAddress, err := config.GetPaymentAddress(orderInfo.PayTokenId)
	if err != nil {
		apiResp.ApiRespErr(http_api.ApiCodeParamsInvalid, err.Error())
		return nil
	}

	resp.OrderId = req.OrderId
	resp.PaymentAddress = paymentAddress
	resp.ContractAddress = orderInfo.PayTokenId.GetContractAddress(config.Cfg.Server.Net)

	apiResp.ApiRespOK(resp)
	return nil
}
