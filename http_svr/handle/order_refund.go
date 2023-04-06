package handle

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"unipay/config"
	"unipay/tables"
)

type ReqOrderRefund struct {
	BusinessId string `json:"business_id"`
	OrderId    string `json:"order_id"`
}

type RespOrderRefund struct {
}

func (h *HttpHandle) OrderRefund(ctx *gin.Context) {
	var (
		funcName             = "OrderRefund"
		clientIp, remoteAddr = GetClientIp(ctx)
		req                  ReqOrderRefund
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

	if err = h.doOrderRefund(&req, &apiResp); err != nil {
		log.Error("doOrderRefund err:", err.Error(), funcName, clientIp, remoteAddr)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doOrderRefund(req *ReqOrderRefund, apiResp *http_api.ApiResp) error {
	var resp RespOrderRefund

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

	// check paid
	if orderInfo.PayStatus != tables.PayStatusPaid {
		apiResp.ApiRespErr(http_api.ApiCodeOrderUnPaid, "order not paid")
		return nil
	}

	// get payment info
	paymentInfo := h.getPaymentInfo(orderInfo.OrderId, apiResp)
	if apiResp.ErrNo != http_api.ApiCodeSuccess {
		return nil
	}

	// update refund status
	if err := h.DbDao.UpdatePaymentInfoToUnRefunded(paymentInfo.PayHash); err != nil {
		apiResp.ApiRespErr(http_api.ApiCodeDbError, "failed to update refund status")
		return fmt.Errorf("UpdatePaymentInfoToUnRefunded err: %s", err.Error())
	}

	apiResp.ApiRespOK(resp)
	return nil
}

func checkBusinessIds(businessId string, apiResp *http_api.ApiResp) {
	if _, ok := config.Cfg.BusinessIds[businessId]; !ok {
		apiResp.ApiRespErr(http_api.ApiCodeParamsInvalid, fmt.Sprintf("unknow bussiness id[%s]", businessId))
	}
}

func (h *HttpHandle) getOrderInfo(orderId, businessId string, apiResp *http_api.ApiResp) (orderInfo tables.TableOrderInfo) {
	var err error
	orderInfo, err = h.DbDao.GetOrderInfo(orderId, businessId)
	if err != nil {
		apiResp.ApiRespErr(http_api.ApiCodeDbError, "failed to get order info")
		log.Error("GetOrderInfo err: ", err.Error())
		return
	}
	if orderInfo.Id == 0 {
		apiResp.ApiRespErr(http_api.ApiCodeOrderNotExist, "order not exist")
		return
	}
	return
}

func (h *HttpHandle) getPaymentInfo(orderId string, apiResp *http_api.ApiResp) (paymentInfo tables.TablePaymentInfo) {
	var err error
	paymentInfo, err = h.DbDao.GetLatestPaymentInfo(orderId)
	if err != nil {
		apiResp.ApiRespErr(http_api.ApiCodeDbError, "failed to get payment info")
		log.Error("GetLatestPaymentInfo err: ", err.Error())
		return
	}
	if paymentInfo.Id == 0 {
		apiResp.ApiRespErr(http_api.ApiCodePaymentNotExist, "payment not exist")
		return
	}
	return
}
