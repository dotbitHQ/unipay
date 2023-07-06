package timer

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/nervosnetwork/ckb-sdk-go/address"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"time"
	"unipay/config"
	"unipay/notify"
)

func (t *ToolTimer) RunCkbBalance() {
	tickerCKBBalance := time.NewTicker(time.Minute * 20)
	t.Wg.Add(1)
	go func() {
		for {
			select {
			case <-tickerCKBBalance.C:
				if err := t.ckbBalance(); err != nil {
					log.Error("ckbBalance err: ", err.Error())
					notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "ckbBalance", err.Error())
				}
			case <-t.Ctx.Done():
				log.Warn("RunCkbBalance done")
				t.Wg.Done()
				return
			}
		}
	}()
}

func (t *ToolTimer) ckbBalance() error {
	for addr, _ := range config.Cfg.Chain.Ckb.AddrMap {
		parseAddr, err := address.Parse(addr)
		if err != nil {
			return fmt.Errorf("address.Parse err: %s", err.Error())
		}
		liveCells, total, err := t.DasCore.GetBalanceCells(&core.ParamGetBalanceCells{
			DasCache:          nil,
			LockScript:        parseAddr.Script,
			CapacityNeed:      0,
			CapacityForChange: 0,
			SearchOrder:       indexer.SearchOrderDesc,
		})
		if err != nil {
			return fmt.Errorf("GetBalanceCells err: %s", err.Error())
		}
		log.Info("ckbBalance:", addr, len(liveCells), total)
		capacity := total / common.OneCkb
		msg := `- Addr: %s
- Count: %d
- Capacity: %d
- Time: %s`
		msg = fmt.Sprintf(msg, addr, len(liveCells), capacity, time.Now().Format("2006-01-02 15:04:05"))

		if capacity < 1000000 {
			notify.SendLarkTextNotifyAtAll(config.Cfg.Notify.LarkDasInfoKey, "CKB Balance", msg)
		} else {
			notify.SendLarkTextNotify(config.Cfg.Notify.LarkDasInfoKey, "CKB Balance", msg)
		}
	}

	return nil
}
