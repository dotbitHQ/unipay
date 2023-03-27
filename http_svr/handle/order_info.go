package handle

import (
	"github.com/dotbitHQ/unipay/http_svr/api_code"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type ReqOrderInfo struct {
}

type RespOrderInfo struct {
}

func (h *HttpHandle) OrderInfo(ctx *gin.Context) {
	var (
		funcName             = "OrderInfo"
		clientIp, remoteAddr = GetClientIp(ctx)
		req                  ReqOrderInfo
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

	if err = h.doOrderInfo(&req, &apiResp); err != nil {
		log.Error("doOrderInfo err:", err.Error(), funcName, clientIp, remoteAddr)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doOrderInfo(req *ReqOrderInfo, apiResp *api_code.ApiResp) error {
	var resp RespOrderInfo

	apiResp.ApiRespOK(resp)
	return nil
}
