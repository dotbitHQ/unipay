package http_svr

import (
	"context"
	"github.com/dotbitHQ/unipay/http_svr/handle"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/mylog"
	"net/http"
)

var (
	log = mylog.NewLogger("http_svr", mylog.LevelDebug)
)

type HttpSvr struct {
	Ctx     context.Context
	Address string
	Engine  *gin.Engine
	Srv     *http.Server
	H       *handle.HttpHandle
}

func (h *HttpSvr) Run() {
	h.initRouter()
	h.Srv = &http.Server{
		Addr:    h.Address,
		Handler: h.Engine,
	}
	go func() {
		if err := h.Srv.ListenAndServe(); err != nil {
			log.Error("ListenAndServe err:", err)
		}
	}()
}

func (h *HttpSvr) Shutdown() {
	if h.Srv != nil {
		log.Warn("HttpSvr Shutdown ... ")
		if err := h.Srv.Shutdown(h.Ctx); err != nil {
			log.Error("Shutdown err:", err.Error())
		}
	}
}
