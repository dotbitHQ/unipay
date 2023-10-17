package timer

import (
	"context"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/http_api/logger"
	"sync"
	"time"
	"unipay/dao"
	"unipay/notify"
)

var (
	log = logger.NewLogger("main", logger.LevelDebug)
)

type ToolTimer struct {
	Ctx     context.Context
	Wg      *sync.WaitGroup
	DbDao   *dao.DbDao
	CN      *notify.CallbackNotice
	DasCore *core.DasCore
}

func (t *ToolTimer) RunCallbackNotice() {
	tickerCallback := time.NewTicker(time.Second * 20)
	t.Wg.Add(1)
	go func() {
		for {
			select {
			case <-tickerCallback.C:
				if err := t.doCallbackNotice(); err != nil {
					log.Error("doCallbackNotice err: ", err.Error())
					notify.SendLarkErrNotify("doCallbackNotice", err.Error())
				}
			case <-t.Ctx.Done():
				log.Warn("RunCallbackNotice done")
				t.Wg.Done()
				return
			}
		}
	}()
}
