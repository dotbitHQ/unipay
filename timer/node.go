package timer

import (
	"fmt"
	"time"
	"unipay/config"
	"unipay/notify"
	"unipay/tables"
)

var (
	nodeMap = make(map[tables.ParserType]uint64)
)

func (t *ToolTimer) RunCheckNode() {
	tickerNode := time.NewTicker(time.Minute * 10)
	t.Wg.Add(1)
	go func() {
		for {
			select {
			case <-tickerNode.C:
				if err := t.doCheckNode(); err != nil {
					log.Error("doCheckNode err: ", err.Error())
					notify.SendLarkErrNotify("doCheckNode", err.Error())
				}
			case <-t.Ctx.Done():
				log.Warn("RunCallbackNotice done")
				t.Wg.Done()
				return
			}
		}
	}()
}

func (t *ToolTimer) doCheckNode() error {
	list, err := t.DbDao.GetLatestNodes()
	if err != nil {
		return fmt.Errorf("GetLatestNodes err: %s", err.Error())
	}
	for _, v := range list {
		if bn, ok := nodeMap[v.ParserType]; ok {
			if v.BlockNumber <= bn {
				msg := fmt.Sprintf("ParserType(%d), BlockNumber[%d,%d]", v.ParserType, v.BlockNumber, bn)
				notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doCheckNode", msg)
			}
		}
		nodeMap[v.ParserType] = v.BlockNumber
		log.Info("doCheckNode:", v.ParserType, v.BlockNumber)
	}
	return nil
}
