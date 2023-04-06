package handle

import (
	"github.com/dotbitHQ/das-lib/http_api"
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

	if err = h.doVersion(&req, &apiResp); err != nil {
		log.Error("doVersion err:", err.Error(), funcName, clientIp, remoteAddr)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doVersion(req *ReqVersion, apiResp *http_api.ApiResp) error {
	var resp RespVersion

	resp.Version = time.Now().String()

	apiResp.ApiRespOK(resp)
	return nil
}
