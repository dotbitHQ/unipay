package handle

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"unipay/tables"
)

type ReqOrderInfo struct {
	BusinessId string `json:"business_id"`
	OrderId    string `json:"order_id"`
}

type RespOrderInfo struct {
	BusinessId    string               `json:"business_id"`
	OrderId       string               `json:"order_id"`
	OrderStatus   tables.OrderStatus   `json:"order_status"`
	PayStatus     tables.PayStatus     `json:"pay_status"`
	PayHash       string               `json:"pay_hash"`
	PayHashStatus tables.PayHashStatus `json:"pay_hash_status"`
	RefundStatus  tables.RefundStatus  `json:"refund_status"`
	RefundHash    string               `json:"refund_hash"`
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

	// get order info
	orderInfo := h.getOrderInfo(req.OrderId, req.BusinessId, apiResp)
	if apiResp.ErrNo != http_api.ApiCodeSuccess {
		return nil
	}

	// get payment info
	paymentInfo, err := h.DbDao.GetLatestPaymentInfo(orderInfo.OrderId)
	if err != nil {
		apiResp.ApiRespErr(http_api.ApiCodeDbError, "failed to get payment info")
		return fmt.Errorf("GetLatestPaymentInfo err: %s", err.Error())
	}

	resp = RespOrderInfo{
		BusinessId:    orderInfo.BusinessId,
		OrderId:       orderInfo.OrderId,
		OrderStatus:   orderInfo.OrderStatus,
		PayStatus:     orderInfo.PayStatus,
		PayHash:       paymentInfo.PayHash,
		PayHashStatus: paymentInfo.PayHashStatus,
		RefundStatus:  paymentInfo.RefundStatus,
		RefundHash:    paymentInfo.RefundHash,
	}

	apiResp.ApiRespOK(resp)
	return nil
}
