package timer

import (
	"context"
	"github.com/dotbitHQ/unipay/dao"
	"github.com/scorpiotzh/mylog"
	"sync"
	"time"
)

var (
	log = mylog.NewLogger("main", mylog.LevelDebug)
)

type ToolTimer struct {
	Ctx   context.Context
	Wg    *sync.WaitGroup
	DbDao *dao.DbDao
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
					// todo notify
				}
			case <-t.Ctx.Done():
				log.Warn("RunCallbackNotice done")
				t.Wg.Done()
				return
			}
		}
	}()
}
