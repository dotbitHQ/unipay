package handle

import (
	"fmt"
	"github.com/dotbitHQ/unipay/config"
	"github.com/dotbitHQ/unipay/http_svr/api_code"
	"github.com/dotbitHQ/unipay/tables"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
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
		apiResp              api_code.ApiResp
		err                  error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddr)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddr, toolib.JsonString(req))

	if err = h.doOrderRefund(&req, &apiResp); err != nil {
		log.Error("doOrderRefund err:", err.Error(), funcName, clientIp, remoteAddr)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doOrderRefund(req *ReqOrderRefund, apiResp *api_code.ApiResp) error {
	var resp RespOrderRefund

	// check business_id
	checkBusinessIds(req.BusinessId, apiResp)
	if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// get order info
	orderInfo, err := h.DbDao.GetOrderInfo(req.OrderId, req.BusinessId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to get order info")
		return fmt.Errorf("GetOrderInfo err: %s", err.Error())
	}
	if orderInfo.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeOrderNotExist, "order not exist")
		return nil
	}

	// check paid
	if orderInfo.PayStatus != tables.PayStatusPaid {
		apiResp.ApiRespErr(api_code.ApiCodeOrderUnPaid, "order un paid")
		return nil
	}

	// get payment info
	paymentInfo, err := h.DbDao.GetUnRefundedPaymentInfo(orderInfo.OrderId, orderInfo.Amount)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to get payment info")
		return fmt.Errorf("GetUnRefundedPaymentInfo err: %s", err.Error())
	}
	if paymentInfo.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodePaymentNotExist, "payment not exist")
		return nil
	}

	// update refund status
	if err := h.DbDao.UpdatePaymentInfoToUnRefunded(paymentInfo.PayHash); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to update refund status")
		return fmt.Errorf("UpdatePaymentInfoToUnRefunded err: %s", err.Error())
	}

	apiResp.ApiRespOK(resp)
	return nil
}

func checkBusinessIds(businessId string, apiResp *api_code.ApiResp) {
	if _, ok := config.Cfg.BusinessIds[businessId]; !ok {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("unknow bussiness id[%s]", businessId))
	}
}
