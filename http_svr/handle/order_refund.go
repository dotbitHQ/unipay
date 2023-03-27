package handle

import (
	"github.com/dotbitHQ/unipay/http_svr/api_code"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type ReqOrderRefund struct {
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

	apiResp.ApiRespOK(resp)
	return nil
}
