package timer

import (
	"fmt"
	"golang.org/x/sync/errgroup"
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

	// callback
	ch := make(chan tables.TableNoticeInfo, 10)
	var errGroup errgroup.Group
	for i := 0; i < 10; i++ {
		errGroup.Go(func() error {
			for notice := range ch {
				if err := t.CN.RepeatCallbackNotice(notice); err != nil {
					log.Error("RepeatCallbackNotice err: ", err.Error(), notice.OrderId)
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
	return nil
}
