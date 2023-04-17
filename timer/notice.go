package timer

import (
	"fmt"
	"golang.org/x/sync/errgroup"
	"sync"
	"unipay/notify"
	"unipay/tables"
)

func (t *ToolTimer) doCallbackNotice() error {
	log.Info("doCallbackNotice start")
	defer log.Info("doCallbackNotice end")

	// get 24h un notify list
	list, err := t.DbDao.Get24HUnNotifyList()
	if err != nil {
		return fmt.Errorf("Get24HUnNotifyList err: %s", err.Error())
	}
	if len(list) == 0 {
		return nil
	}
	log.Info("doCallbackNotice list:", len(list))

	// callback
	ch := make(chan tables.TableNoticeInfo, 10)
	var eventMap = make(map[string][]notify.EventInfo)
	var lock sync.Mutex
	var errGroup errgroup.Group
	for i := 0; i < 10; i++ {
		errGroup.Go(func() error {
			for notice := range ch {
				businessId, eventInfo, er := t.CN.GetEventInfo(notice)
				if er != nil {
					log.Error("GetEventInfo err: ", er.Error(), notice.OrderId)
					continue
				}
				if businessId != "" && eventInfo.OrderId != "" {
					lock.Lock()
					eventMap[businessId] = append(eventMap[businessId], eventInfo)
					lock.Unlock()
				}
			}
			return nil
		})
	}

	// range list
	for i := range list {
		ch <- list[i]
	}
	close(ch)

	if err = errGroup.Wait(); err != nil {
		return fmt.Errorf("errGroup.Wait() err: %s", err.Error())
	}
	// callback
	if err := t.CN.RepeatCallbackNotice(eventMap); err != nil {
		return fmt.Errorf("RepeatCallbackNotice err: %s", err.Error())
	}

	return nil
}
