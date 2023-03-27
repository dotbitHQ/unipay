package handle

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/unipay/config"
	"github.com/dotbitHQ/unipay/http_svr/api_code"
	"github.com/dotbitHQ/unipay/tables"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"net/http"
	"time"
)

type ReqOrderCreate struct {
	core.ChainTypeAddress
	BusinessId string            `json:"business_id"`
	Amount     decimal.Decimal   `json:"amount"`
	PayTokenId tables.PayTokenId `json:"pay_token_id"`
}

type RespOrderCreate struct {
	OrderId        string `json:"order_id"`
	PaymentAddress string `json:"payment_address"`
}

func (h *HttpHandle) OrderCreate(ctx *gin.Context) {
	var (
		funcName             = "OrderCreate"
		clientIp, remoteAddr = GetClientIp(ctx)
		req                  ReqOrderCreate
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

	if err = h.doOrderCreate(&req, &apiResp); err != nil {
		log.Error("doOrderCreate err:", err.Error(), funcName, clientIp, remoteAddr)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doOrderCreate(req *ReqOrderCreate, apiResp *api_code.ApiResp) error {
	var resp RespOrderCreate

	// check key
	addrHex, err := req.FormatChainTypeAddress(h.DasCore.NetType(), true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, err.Error())
		return fmt.Errorf("FormatChainTypeAddress err: %s", err.Error())
	}

	// check business_id
	if _, ok := config.Cfg.BusinessIds[req.BusinessId]; !ok {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("unknow bussiness id[%s]", req.BusinessId))
		return nil
	}

	// check pay token id
	paymentAddress, err := config.GetPaymentAddress(req.PayTokenId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, err.Error())
		return nil
	}
	log.Info("doOrderCreate:", paymentAddress, req.PayTokenId)

	// create order
	orderInfo := tables.TableOrderInfo{
		OrderId:     "",
		BusinessId:  req.BusinessId,
		PayAddress:  addrHex.AddressHex,
		AlgorithmId: addrHex.DasAlgorithmId,
		Amount:      req.Amount,
		PayTokenId:  req.PayTokenId,
		PayStatus:   tables.PayStatusUnpaid,
		OrderStatus: tables.OrderStatusNormal,
		Timestamp:   time.Now().UnixNano() / 1e6,
	}
	orderInfo.InitOrderId()

	if err := h.DbDao.CreateOrder(orderInfo); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to create order")
		return fmt.Errorf("CreateOrder err: %s", err.Error())
	}
	//
	resp.OrderId = orderInfo.OrderId
	resp.PaymentAddress = paymentAddress

	apiResp.ApiRespOK(resp)
	return nil
}
