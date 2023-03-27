package handle

import (
	"github.com/dotbitHQ/unipay/http_svr/api_code"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"time"
)

type ReqVersion struct {
}

type RespVersion struct {
	Version string `json:"version"`
}

func (h *HttpHandle) Version(ctx *gin.Context) {
	var (
		funcName             = "Version"
		clientIp, remoteAddr = GetClientIp(ctx)
		req                  ReqVersion
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

	if err = h.doVersion(&req, &apiResp); err != nil {
		log.Error("doVersion err:", err.Error(), funcName, clientIp, remoteAddr)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doVersion(req *ReqVersion, apiResp *api_code.ApiResp) error {
	var resp RespVersion

	resp.Version = time.Now().String()

	apiResp.ApiRespOK(resp)
	return nil
}
