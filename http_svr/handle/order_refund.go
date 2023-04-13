package handle

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"unipay/config"
)

type RefundInfo struct {
	OrderId string `json:"order_id"`
	PayHash string `json:"pay_hash"`
}

type ReqOrderRefund struct {
	BusinessId string       `json:"business_id"`
	RefundList []RefundInfo `json:"refund_list"`
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

	var payHashList []string
	var refundMap = make(map[string]string)
	for _, v := range req.RefundList {
		payHashList = append(payHashList, v.PayHash)
		refundMap[v.PayHash] = v.OrderId
	}

	// get payment info
	paymentList, err := h.DbDao.GetPaymentByPayHashListWithStatus(payHashList)
	if err != nil {
		apiResp.ApiRespErr(http_api.ApiCodeDbError, "failed to search payment")
		return fmt.Errorf("GetPaymentByPayHashList err: %s", err.Error())
	}
	if len(paymentList) == 0 {
		apiResp.ApiRespOK(resp)
		return nil
	}

	// update refund status
	for _, v := range paymentList {
		if orderId, ok := refundMap[v.PayHash]; !ok || orderId != v.OrderId {
			continue
		}
		if err := h.DbDao.UpdatePaymentInfoToUnRefunded(v.PayHash); err != nil {
			log.Error("UpdatePaymentInfoToUnRefunded err: %s", err.Error())
		}
	}

	apiResp.ApiRespOK(resp)
	return nil
}

func checkBusinessIds(businessId string, apiResp *http_api.ApiResp) {
	if _, ok := config.Cfg.BusinessIds[businessId]; !ok {
		apiResp.ApiRespErr(http_api.ApiCodeParamsInvalid, fmt.Sprintf("unknow bussiness id[%s]", businessId))
	}
}
