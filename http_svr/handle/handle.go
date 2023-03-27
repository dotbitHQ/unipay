package handle

import (
	"context"
	"fmt"
	"github.com/dotbitHQ/unipay/dao"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/mylog"
)

var (
	log = mylog.NewLogger("http_handle", mylog.LevelDebug)
)

type HttpHandle struct {
	Ctx   context.Context
	DbDao *dao.DbDao
}

func GetClientIp(ctx *gin.Context) (string, string) {
	clientIP := fmt.Sprintf("%v", ctx.Request.Header.Get("X-Real-IP"))
	return clientIP, ctx.Request.RemoteAddr
}
