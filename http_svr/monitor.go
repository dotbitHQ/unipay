package http_svr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/parnurzeal/gorequest"
	"net/http"
	"time"
	"unipay/txtool"
)

type ReqPushLog struct {
	Index   string        `json:"index"`
	Method  string        `json:"method"`
	Ip      string        `json:"ip"`
	Latency time.Duration `json:"latency"`
	ErrMsg  string        `json:"err_msg"`
	ErrNo   int           `json:"err_no"`
}

func PushLog(url string, req ReqPushLog) {
	if url == "" {
		return
	}
	go func() {
		resp, _, errs := gorequest.New().Post(url).SendStruct(&req).End()
		if len(errs) > 0 {
			log.Error("PushLog err:", errs)
		} else if resp.StatusCode != http.StatusOK {
			log.Error("PushLog StatusCode:", resp.StatusCode)
		}
	}()
}

func DoMonitorLog(method string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startTime := time.Now()

		blw := &bodyWriter{body: bytes.NewBufferString(""), ResponseWriter: ctx.Writer}
		ctx.Writer = blw
		ctx.Next()
		statusCode := ctx.Writer.Status()

		var resp http_api.ApiResp
		if statusCode == http.StatusOK && blw.body.String() != "" {
			if err := json.Unmarshal(blw.body.Bytes(), &resp); err != nil {
				log.Warn("DoMonitorLog Unmarshal err:", method, err)
				return
			}
			if resp.ErrNo != http_api.ApiCodeSuccess {
				log.Warn("DoMonitorLog:", method, resp.ErrNo, resp.ErrMsg)
			}
		}
		txtool.Tools.Metrics.Api().WithLabelValues(method, fmt.Sprint(statusCode), fmt.Sprint(resp.ErrNo), resp.ErrMsg).Observe(time.Since(startTime).Seconds())
	}
}

func getClientIp(ctx *gin.Context) string {
	return fmt.Sprintf("%v", ctx.Request.Header.Get("X-Real-IP"))
}

type bodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (b bodyWriter) Write(bys []byte) (int, error) {
	b.body.Write(bys)
	return b.ResponseWriter.Write(bys)
}
