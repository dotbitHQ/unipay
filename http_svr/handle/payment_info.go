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
	OrderIdList []string `json:"order_id_list"` // for check pay status
	PayHashList []string `json:"pay_hash_list"` // for check refund status
}

type RespPaymentInfo struct {
	PaymentList []PaymentInfo `json:"payment_list"`
}

type PaymentInfo struct {
	OrderId         string               `json:"order_id"`
	PayHash         string               `json:"pay_hash"`
	PayHashStatus   tables.PayHashStatus `json:"pay_hash_status"`
	RefundHash      string               `json:"refund_hash"`
	RefundStatus    tables.RefundStatus  `json:"refund_status"`
	PaymentAddress  string               `json:"payment_address"`
	ContractAddress string               `json:"contract_address"`
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

	// get payment list by order id
	list, err := h.DbDao.GetPaymentListByOrderIds(req.OrderIdList)
	if err != nil {
		apiResp.ApiRespErr(http_api.ApiCodeDbError, "failed to get payment info")
		return fmt.Errorf("GetPaymentListByOrderIds err: %s", err.Error())
	}
	var paymentMap = make(map[string]PaymentInfo)
	for _, v := range list {
		paymentMap[v.PayHash] = PaymentInfo{
			OrderId:       v.OrderId,
			PayHash:       v.PayHash,
			PayHashStatus: v.PayHashStatus,
			RefundHash:    v.RefundHash,
			RefundStatus:  v.RefundStatus,
		}
	}

	// get payment list by pay hash
	list, err = h.DbDao.GetPaymentByPayHashList(req.PayHashList)
	if err != nil {
		apiResp.ApiRespErr(http_api.ApiCodeDbError, "failed to get payment info")
		return fmt.Errorf("GetPaymentByPayHashList err: %s", err.Error())
	}
	for _, v := range list {
		paymentMap[v.PayHash] = PaymentInfo{
			OrderId:       v.OrderId,
			PayHash:       v.PayHash,
			PayHashStatus: v.PayHashStatus,
			RefundHash:    v.RefundHash,
			RefundStatus:  v.RefundStatus,
		}
	}

	for k := range paymentMap {
		resp.PaymentList = append(resp.PaymentList, paymentMap[k])
	}

	apiResp.ApiRespOK(resp)
	return nil
}
