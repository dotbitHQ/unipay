package handle

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"net/http"
	"time"
	"unipay/config"
	"unipay/stripe_api"
	"unipay/tables"
)

type ReqOrderCreate struct {
	core.ChainTypeAddress
	BusinessId string            `json:"business_id"`
	Amount     decimal.Decimal   `json:"amount"`
	PayTokenId tables.PayTokenId `json:"pay_token_id"`
}

type RespOrderCreate struct {
	OrderId         string `json:"order_id"`
	PaymentAddress  string `json:"payment_address"`
	ContractAddress string `json:"contract_address"`
	ClientSecret    string `json:"client_secret"`
}

func (h *HttpHandle) OrderCreate(ctx *gin.Context) {
	var (
		funcName             = "OrderCreate"
		clientIp, remoteAddr = GetClientIp(ctx)
		req                  ReqOrderCreate
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

	if err = h.doOrderCreate(&req, &apiResp); err != nil {
		log.Error("doOrderCreate err:", err.Error(), funcName, clientIp, remoteAddr)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doOrderCreate(req *ReqOrderCreate, apiResp *http_api.ApiResp) error {
	var resp RespOrderCreate

	// check key
	addrHex, err := req.FormatChainTypeAddress(h.DasCore.NetType(), true)
	if err != nil {
		apiResp.ApiRespErr(http_api.ApiCodeParamsInvalid, err.Error())
		return fmt.Errorf("FormatChainTypeAddress err: %s", err.Error())
	}

	// check business_id
	checkBusinessIds(req.BusinessId, apiResp)
	if apiResp.ErrNo != http_api.ApiCodeSuccess {
		return nil
	}

	// check pay token id
	paymentAddress, err := config.GetPaymentAddress(req.PayTokenId)
	if err != nil {
		apiResp.ApiRespErr(http_api.ApiCodeParamsInvalid, err.Error())
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
		Timestamp:   time.Now().UnixMilli(),
	}
	orderInfo.InitOrderId()
	var paymentInfo tables.TablePaymentInfo
	if req.PayTokenId == tables.PayTokenIdStripeUSD {
		if req.Amount.IntPart() < 50 {
			apiResp.ApiRespErr(http_api.ApiCodeAmountIsTooLow, "Amount not less than 0.5$")
			return nil
		}
		pi, err := stripe_api.CreatePaymentIntent(orderInfo.OrderId, req.Amount.IntPart())
		if err != nil {
			apiResp.ApiRespErr(http_api.ApiCodeError500, "Failed to create a payment intent")
			return fmt.Errorf("CreatePaymentIntent err: %s", err.Error())
		}
		paymentInfo = tables.TablePaymentInfo{
			PayHash:     pi.ID,
			OrderId:     orderInfo.OrderId,
			PayAddress:  orderInfo.PayAddress,
			AlgorithmId: orderInfo.AlgorithmId,
			Timestamp:   time.Now().UnixMilli(),
			Amount:      req.Amount,
			PayTokenId:  req.PayTokenId,
		}
		resp.ClientSecret = pi.ClientSecret
	}
	if err := h.DbDao.CreateOrderInfoWithPaymentInfo(orderInfo, paymentInfo); err != nil {
		apiResp.ApiRespErr(http_api.ApiCodeDbError, "Failed to create order")
		return fmt.Errorf("CreateOrderInfoWithPaymentInfo err: %s", err.Error())
	}

	//
	resp.OrderId = orderInfo.OrderId
	resp.PaymentAddress = paymentAddress
	resp.ContractAddress = req.PayTokenId.GetContractAddress(config.Cfg.Server.Net)

	apiResp.ApiRespOK(resp)
	return nil
}
