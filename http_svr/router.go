package http_svr

import (
	"encoding/json"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

func (h *HttpSvr) initRouter() {
	h.engine.Use(toolib.MiddlewareCors())
	v1 := h.engine.Group("v1")
	{
		// cache
		//shortExpireTime, longExpireTime, lockTime := time.Second*5, time.Second*15, time.Minute
		//shortDataTime, longDataTime := time.Minute*3, time.Minute*10
		//cacheHandleShort := toolib.MiddlewareCacheByRedis(h.rc.GetRedisClient(), false, shortDataTime, lockTime, shortExpireTime, respHandle)
		//cacheHandleLong := toolib.MiddlewareCacheByRedis(h.rc.GetRedisClient(), false, longDataTime, lockTime, longExpireTime, respHandle)
		//cacheHandleShortCookies := toolib.MiddlewareCacheByRedis(h.rc.GetRedisClient(), true, shortDataTime, lockTime, shortExpireTime, respHandle)

		// query
		v1.POST("/version", DoMonitorLog("version"), h.H.Version)
		v1.POST("/order/create", DoMonitorLog("order_create"), h.H.OrderCreate)
		v1.POST("/order/refund", DoMonitorLog("order_refund"), h.H.OrderRefund)
		v1.POST("/order/info", DoMonitorLog("order_info"), h.H.OrderInfo)

		// operate
	}
}

func respHandle(c *gin.Context, res string, err error) {
	if err != nil {
		log.Error("respHandle err:", err.Error())
		c.AbortWithStatusJSON(http.StatusOK, http_api.ApiRespErr(http.StatusInternalServerError, err.Error()))
	} else if res != "" {
		var respMap map[string]interface{}
		_ = json.Unmarshal([]byte(res), &respMap)
		c.AbortWithStatusJSON(http.StatusOK, respMap)
	}
}
