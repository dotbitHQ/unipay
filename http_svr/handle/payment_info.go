package handle

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"unipay/tables"
)

type ReqPaymentInfo struct {
	BusinessId  string   `json:"business_id"`
	PayHashList []string `json:"pay_hash_list"`
}

type RespPaymentInfo struct {
	BusinessId  string        `json:"business_id"`
	PaymentList []PaymentInfo `json:"payment_list"`
}

type PaymentInfo struct {
	OrderId       string               `json:"order_id"`
	PayHash       string               `json:"pay_hash"`
	PayHashStatus tables.PayHashStatus `json:"pay_hash_status"`
	RefundStatus  tables.RefundStatus  `json:"refund_status"`
	RefundHash    string               `json:"refund_hash"`
}

func (h *HttpHandle) PaymentInfo(ctx *gin.Context) {
	var (
		funcName             = "PaymentInfo"
		clientIp, remoteAddr = GetClientIp(ctx)
		req                  ReqPaymentInfo
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

	if err = h.doPaymentInfo(&req, &apiResp); err != nil {
		log.Error("doPaymentInfo err:", err.Error(), funcName, clientIp, remoteAddr)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doPaymentInfo(req *ReqPaymentInfo, apiResp *http_api.ApiResp) error {
	var resp RespPaymentInfo
	resp.PaymentList = make([]PaymentInfo, 0)

	// check business_id
	checkBusinessIds(req.BusinessId, apiResp)
	if apiResp.ErrNo != http_api.ApiCodeSuccess {
		return nil
	}

	// get payment info
	list, err := h.DbDao.GetPaymentByPayHashList(req.PayHashList)
	if err != nil {
		apiResp.ApiRespErr(http_api.ApiCodeDbError, "failed to get payment info")
		return fmt.Errorf("GetPaymentByPayHashList err: %s", err.Error())
	}
	var paymentMap = make(map[string]tables.TablePaymentInfo)
	for i, v := range list {
		paymentMap[v.PayHash] = list[i]
	}

	//
	resp.BusinessId = req.BusinessId
	for _, v := range req.PayHashList {
		item, ok := paymentMap[v]
		if !ok {
			continue
		}
		paymentInfo := PaymentInfo{
			OrderId:       item.OrderId,
			PayHash:       item.PayHash,
			PayHashStatus: item.PayHashStatus,
			RefundStatus:  item.RefundStatus,
			RefundHash:    item.RefundHash,
		}
		resp.PaymentList = append(resp.PaymentList, paymentInfo)
	}

	apiResp.ApiRespOK(resp)
	return nil
}
