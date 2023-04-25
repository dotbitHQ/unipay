package refund

import (
	"context"
	"fmt"
	"github.com/dotbitHQ/das-lib/bitcoin"
	"github.com/dotbitHQ/das-lib/chain/chain_evm"
	"github.com/dotbitHQ/das-lib/chain/chain_tron"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/remote_sign"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/robfig/cron/v3"
	"github.com/scorpiotzh/mylog"
	"sync"
	"time"
	"unipay/config"
	"unipay/dao"
)

var (
	log = mylog.NewLogger("refund", mylog.LevelDebug)
)

type ToolRefund struct {
	Ctx           context.Context
	Wg            *sync.WaitGroup
	DbDao         *dao.DbDao
	DasCore       *core.DasCore
	TxBuilderBase *txbuilder.DasTxBuilderBase

	remoteSignClient *remote_sign.RemoteSignClient
	chainDoge        *bitcoin.TxTool
	chainEth         *chain_evm.ChainEvm
	chainBsc         *chain_evm.ChainEvm
	chainPolygon     *chain_evm.ChainEvm
	chainTron        *chain_tron.ChainTron

	cron *cron.Cron
}

func (t *ToolRefund) InitRefundInfo() error {
	// remote sign client
	remoteSignClient, err := remote_sign.NewRemoteSignClient(t.Ctx, config.Cfg.Server.RemoteSignApiUrl)
	if err != nil {
		return fmt.Errorf("NewRemoteSignClient err: %s", err.Error())
	}
	t.remoteSignClient = remoteSignClient
	// doge
	t.chainDoge = &bitcoin.TxTool{
		RpcClient: &bitcoin.BaseRequest{
			RpcUrl:   config.Cfg.Chain.Doge.Node,
			User:     config.Cfg.Chain.Doge.User,
			Password: config.Cfg.Chain.Doge.Password,
			Proxy:    "",
		},
		Ctx:              t.Ctx,
		RemoteSignClient: remoteSignClient.Client(),
		DustLimit:        bitcoin.DustLimitDoge,
		Params:           bitcoin.GetDogeMainNetParams(),
	}

	// eth
	chainEth, err := chain_evm.NewChainEvm(t.Ctx, config.Cfg.Chain.Eth.Node, config.Cfg.Chain.Eth.RefundAddFee)
	if err != nil {
		return fmt.Errorf("NewChainEvm eth err: %s", err.Error())
	}
	t.chainEth = chainEth

	//bsc
	chainBsc, err := chain_evm.NewChainEvm(t.Ctx, config.Cfg.Chain.Bsc.Node, config.Cfg.Chain.Bsc.RefundAddFee)
	if err != nil {
		return fmt.Errorf("NewChainEvm bsc err: %s", err.Error())
	}
	t.chainBsc = chainBsc

	//polygon
	chainPolygon, err := chain_evm.NewChainEvm(t.Ctx, config.Cfg.Chain.Polygon.Node, config.Cfg.Chain.Polygon.RefundAddFee)
	if err != nil {
		return fmt.Errorf("NewChainEvm polygon err: %s", err.Error())
	}
	t.chainPolygon = chainPolygon
	return nil
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

func (t *ToolRefund) RunRefundOnce() {
	tickerNow := time.NewTicker(time.Second * 10)
	go func() {
		select {
		case <-tickerNow.C:
			log.Info("RunRefundOnce start")
			if err := t.doRefund(); err != nil {
				log.Error("doRefund err: %s", err.Error())
			}
			log.Info("RunRefundOnce end")
		case <-t.Ctx.Done():
			log.Info("RunRefundOnce done")
			return
		}
	}()
}
