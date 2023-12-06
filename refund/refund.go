package refund

import (
	"context"
	"fmt"
	"github.com/dotbitHQ/das-lib/bitcoin"
	"github.com/dotbitHQ/das-lib/chain/chain_evm"
	"github.com/dotbitHQ/das-lib/chain/chain_tron"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/http_api/logger"
	"github.com/dotbitHQ/das-lib/remote_sign"
	"github.com/robfig/cron/v3"
	"sync"
	"time"
	"unipay/config"
	"unipay/dao"
)

var (
	log = logger.NewLogger("refund", logger.LevelDebug)
)

type ToolRefund struct {
	Ctx     context.Context
	Wg      *sync.WaitGroup
	DbDao   *dao.DbDao
	DasCore *core.DasCore

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
	if config.Cfg.Server.RemoteSignApiUrl != "" {
		remoteSignClient, err := remote_sign.NewRemoteSignClient(t.Ctx, config.Cfg.Server.RemoteSignApiUrl)
		if err != nil {
			return fmt.Errorf("NewRemoteSignClient err: %s", err.Error())
		}
		t.remoteSignClient = remoteSignClient
	}
	// doge
	if config.Cfg.Chain.Doge.Refund {
		t.chainDoge = &bitcoin.TxTool{
			RpcClient: &bitcoin.BaseRequest{
				RpcUrl:   config.Cfg.Chain.Doge.Node,
				User:     config.Cfg.Chain.Doge.User,
				Password: config.Cfg.Chain.Doge.Password,
				Proxy:    "",
			},
			Ctx:              t.Ctx,
			RemoteSignClient: nil,
			DustLimit:        bitcoin.DustLimitDoge,
			Params:           bitcoin.GetDogeMainNetParams(),
		}
		if t.remoteSignClient != nil {
			t.chainDoge.RemoteSignClient = t.remoteSignClient.Client()
		}
	}

	// eth
	if config.Cfg.Chain.Eth.Refund {
		chainEth, err := chain_evm.NewChainEvm(t.Ctx, config.Cfg.Chain.Eth.Node, config.Cfg.Chain.Eth.RefundAddFee)
		if err != nil {
			return fmt.Errorf("NewChainEvm eth err: %s", err.Error())
		}
		t.chainEth = chainEth
	}

	// bsc
	if config.Cfg.Chain.Bsc.Refund {
		chainBsc, err := chain_evm.NewChainEvm(t.Ctx, config.Cfg.Chain.Bsc.Node, config.Cfg.Chain.Bsc.RefundAddFee)
		if err != nil {
			return fmt.Errorf("NewChainEvm bsc err: %s", err.Error())
		}
		t.chainBsc = chainBsc
	}

	// polygon
	if config.Cfg.Chain.Polygon.Refund {
		chainPolygon, err := chain_evm.NewChainEvm(t.Ctx, config.Cfg.Chain.Polygon.Node, config.Cfg.Chain.Polygon.RefundAddFee)
		if err != nil {
			return fmt.Errorf("NewChainEvm polygon err: %s", err.Error())
		}
		t.chainPolygon = chainPolygon
	}

	// tron
	if config.Cfg.Chain.Tron.Refund {
		chainTron, err := chain_tron.NewChainTron(t.Ctx, config.Cfg.Chain.Tron.Node)
		if err != nil {
			return fmt.Errorf("chain_ckb.NewChainTron tron err: %s", err.Error())
		}
		t.chainTron = chainTron
	}

	return nil
}

func (t *ToolRefund) RunRefund() error {
	if config.Cfg.Server.CronSpec == "" {
		return nil
	}
	log.Debug("DoOrderRefund:", config.Cfg.Server.CronSpec)

	t.cron = cron.New(cron.WithSeconds())
	_, err := t.cron.AddFunc(config.Cfg.Server.CronSpec, func() {
		log.Debug("doRefund start ...")
		if err := t.doRefund(); err != nil {
			log.Error("doRefund err: ", err.Error())
		}
		log.Debug("doRefund end ...")
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
