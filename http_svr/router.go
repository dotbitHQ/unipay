package http_svr

import (
	"encoding/json"
	"github.com/dotbitHQ/unipay/http_svr/api_code"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

func (h *HttpSvr) initRouter() {
	h.Engine.Use(toolib.MiddlewareCors())
	v1 := h.Engine.Group("v1")
	{
		// cache
		//shortExpireTime, longExpireTime, lockTime := time.Second*5, time.Second*15, time.Minute
		//shortDataTime, longDataTime := time.Minute*3, time.Minute*10
		//cacheHandleShort := toolib.MiddlewareCacheByRedis(h.rc.GetRedisClient(), false, shortDataTime, lockTime, shortExpireTime, respHandle)
		//cacheHandleLong := toolib.MiddlewareCacheByRedis(h.rc.GetRedisClient(), false, longDataTime, lockTime, longExpireTime, respHandle)
		//cacheHandleShortCookies := toolib.MiddlewareCacheByRedis(h.rc.GetRedisClient(), true, shortDataTime, lockTime, shortExpireTime, respHandle)

		// query
		v1.POST("/version", api_code.DoMonitorLog("Version"), h.H.Version)

		// operate
	}
}

func respHandle(c *gin.Context, res string, err error) {
	if err != nil {
		log.Error("respHandle err:", err.Error())
		c.AbortWithStatusJSON(http.StatusOK, api_code.ApiRespErr(http.StatusInternalServerError, err.Error()))
	} else if res != "" {
		var respMap map[string]interface{}
		_ = json.Unmarshal([]byte(res), &respMap)
		c.AbortWithStatusJSON(http.StatusOK, respMap)
	}
}
