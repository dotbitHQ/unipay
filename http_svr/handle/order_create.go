package handle

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
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
	BusinessId        string            `json:"business_id"`
	Amount            decimal.Decimal   `json:"amount"`
	PayTokenId        tables.PayTokenId `json:"pay_token_id"`
	PaymentAddress    string            `json:"payment_address"`
	PremiumPercentage decimal.Decimal   `json:"premium_percentage"`
	PremiumBase       decimal.Decimal   `json:"premium_base"`
	PremiumAmount     decimal.Decimal   `json:"premium_amount"`
	MetaData          map[string]string `json:"meta_data"`
}

type RespOrderCreate struct {
	OrderId               string `json:"order_id"`
	PaymentAddress        string `json:"payment_address"`
	ContractAddress       string `json:"contract_address"`
	StripePaymentIntentId string `json:"stripe_payment_intent_id"`
	ClientSecret          string `json:"client_secret"`
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

	// create order
	orderInfo := tables.TableOrderInfo{
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
	if orderInfo.Amount.LessThanOrEqual(decimal.Zero) {
		payHash := common.Bytes2Hex(common.Blake2b([]byte(orderInfo.OrderId)))
		paymentInfo = tables.TablePaymentInfo{
			OrderId:       orderInfo.OrderId,
			PayHash:       payHash,
			PayAddress:    addrHex.AddressHex,
			AlgorithmId:   addrHex.DasAlgorithmId,
			Timestamp:     time.Now().UnixMilli(),
			Amount:        orderInfo.Amount,
			PayTokenId:    orderInfo.PayTokenId,
			PayHashStatus: tables.PayHashStatusPending,
			RefundStatus:  tables.RefundStatusDefault,
		}
		noticeInfo := tables.TableNoticeInfo{
			EventType:    tables.EventTypeOrderPay,
			PayHash:      paymentInfo.PayHash,
			NoticeStatus: tables.NoticeStatusDefault,
			Timestamp:    time.Now().UnixMilli(),
		}
		noticeInfo.InitNoticeId()
		if err := h.DbDao.CreateOrderInfNoNeedPay(orderInfo, paymentInfo, noticeInfo); err != nil {
			apiResp.ApiRespErr(http_api.ApiCodeDbError, "Failed to create order")
			return fmt.Errorf("CreateOrderInfoWithPaymentInfo err: %s", err.Error())
		}

		resp.OrderId = orderInfo.OrderId
		resp.PaymentAddress = req.PaymentAddress
		resp.ContractAddress = req.PayTokenId.GetContractAddress(config.Cfg.Server.Net)
		apiResp.ApiRespOK(resp)
		return nil
	}

	// check pay token id
	paymentAddress, err := config.GetPaymentAddress(req.PayTokenId, req.PaymentAddress)
	if err != nil {
		apiResp.ApiRespErr(http_api.ApiCodeParamsInvalid, err.Error())
		return nil
	}
	orderInfo.PaymentAddress = paymentAddress
	log.Info("doOrderCreate:", paymentAddress, req.PayTokenId)

	if req.PayTokenId == tables.PayTokenIdStripeUSD {
		if !config.Cfg.Chain.Stripe.Switch {
			apiResp.ApiRespErr(http_api.ApiCodePaymentMethodDisable, "This payment method is unavailable")
			return nil
		}
		if req.Amount.IntPart() < 52 {
			apiResp.ApiRespErr(http_api.ApiCodeAmountIsTooLow, "Amount not less than 0.52$")
			return nil
		}

		orderInfo.PremiumPercentage = req.PremiumPercentage
		orderInfo.PremiumBase = req.PremiumBase
		orderInfo.PremiumAmount = req.PremiumAmount

		pi, err := stripe_api.CreatePaymentIntent(req.BusinessId, orderInfo.OrderId, req.MetaData, req.Amount.IntPart())
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
		resp.StripePaymentIntentId = pi.ID
		resp.ClientSecret = pi.ClientSecret
	}

	if err := h.DbDao.CreateOrderInfoWithPaymentInfo(orderInfo, paymentInfo); err != nil {
		apiResp.ApiRespErr(http_api.ApiCodeDbError, "Failed to create order")
		return fmt.Errorf("CreateOrderInfoWithPaymentInfo err: %s", err.Error())
	}

	resp.OrderId = orderInfo.OrderId
	resp.PaymentAddress = req.PaymentAddress
	resp.ContractAddress = req.PayTokenId.GetContractAddress(config.Cfg.Server.Net)

	apiResp.ApiRespOK(resp)
	return nil
}
