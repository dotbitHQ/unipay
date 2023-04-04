package http_svr

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/mylog"
	"net/http"
	"unipay/http_svr/handle"
)

var (
	log = mylog.NewLogger("http_svr", mylog.LevelDebug)
)

type HttpSvr struct {
	Ctx     context.Context
	Address string
	H       *handle.HttpHandle
	engine  *gin.Engine
	srv     *http.Server
}

func (h *HttpSvr) Run() {
	h.engine = gin.New()
	h.initRouter()
	h.srv = &http.Server{
		Addr:    h.Address,
		Handler: h.engine,
	}
	go func() {
		if err := h.srv.ListenAndServe(); err != nil {
			log.Error("ListenAndServe err:", err)
		}
	}()
}

func (h *HttpSvr) Shutdown() {
	if h.srv != nil {
		log.Warn("HttpSvr Shutdown ... ")
		if err := h.srv.Shutdown(h.Ctx); err != nil {
			log.Error("Shutdown err:", err.Error())
		}
	}
}
