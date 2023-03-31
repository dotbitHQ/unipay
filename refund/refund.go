package refund

import (
	"context"
	"fmt"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/unipay/config"
	"github.com/dotbitHQ/unipay/dao"
	"github.com/dotbitHQ/unipay/parser"
	"github.com/robfig/cron/v3"
	"github.com/scorpiotzh/mylog"
	"sync"
)

var (
	log = mylog.NewLogger("refund", mylog.LevelDebug)
)

type ToolRefund struct {
	Ctx        context.Context
	Wg         *sync.WaitGroup
	DbDao      *dao.DbDao
	DasCore    *core.DasCore
	ToolParser *parser.ToolParser

	cron *cron.Cron
}

func (t *ToolRefund) RunRefund() error {
	if config.Cfg.Server.CronSpec == "" {
		return nil
	}
	log.Info("DoOrderRefund:", config.Cfg.Server.CronSpec)

	t.cron = cron.New(cron.WithSeconds())
	_, err := t.cron.AddFunc(config.Cfg.Server.CronSpec, func() {
		log.Info("doRefund start ...")
		if err := t.doRefund(); err != nil {
			log.Error("doRefund err: ", err.Error())
		}
		log.Info("doRefund end ...")
	})
	if err != nil {
		return fmt.Errorf("c.AddFunc err: %s", err.Error())
	}
	t.cron.Start()
	return nil
}
