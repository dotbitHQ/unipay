package handle

import (
	"context"
	"fmt"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/http_api/logger"
	"github.com/gin-gonic/gin"
	"unipay/dao"
	"unipay/notify"
)

var (
	log = logger.NewLogger("http_handle", logger.LevelDebug)
)

type HttpHandle struct {
	Ctx     context.Context
	DbDao   *dao.DbDao
	DasCore *core.DasCore
	CN      *notify.CallbackNotice
}

func GetClientIp(ctx *gin.Context) (string, string) {
	clientIP := fmt.Sprintf("%v", ctx.Request.Header.Get("X-Real-IP"))
	return clientIP, ctx.Request.RemoteAddr
}
