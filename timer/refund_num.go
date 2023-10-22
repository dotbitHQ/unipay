package timer

import (
	"fmt"
	"time"
	"unipay/notify"
)

func (t *ToolTimer) RunCheckRefundNum() {
	tickerCheck := time.NewTicker(time.Minute * 30)
	t.Wg.Add(1)
	go func() {
		for {
			select {
			case <-tickerCheck.C:
				if err := t.checkRefundNum(); err != nil {
					log.Error("checkRefundNum err: ", err.Error())
					notify.SendLarkErrNotify("checkRefundNum", err.Error())
				}
			case <-t.Ctx.Done():
				log.Warn("RunCheckRefundNum done")
				t.Wg.Done()
				return
			}
		}
	}()
}

func (t *ToolTimer) checkRefundNum() error {
	countRefund, err := t.DbDao.GetUnRefundTxCount()
	if err != nil {
		return fmt.Errorf("GetUnRefundTxCount err: %s", err.Error())
	}
	if countRefund == 0 {
		return nil
	}
	log.Info("checkRefundNum:", countRefund)
	msg := fmt.Sprintf("> count: %d", countRefund)
	notify.SendLarkErrNotify("UnRefunded Txs", msg)
	return nil
}
