package main

import (
	"context"
	"fmt"
	"github.com/scorpiotzh/mylog"
	"github.com/scorpiotzh/toolib"
	"github.com/urfave/cli/v2"
	"os"
	"sync"
	"time"
	"unipay/config"
	"unipay/dao"
	"unipay/refund"
)

var (
	log               = mylog.NewLogger("main", mylog.LevelDebug)
	exit              = make(chan struct{})
	ctxServer, cancel = context.WithCancel(context.Background())
	wgServer          = sync.WaitGroup{}
)

func main() {
	log.Debugf("server start：")
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Load configuration from `FILE`",
			},
		},
		Action: runServer,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func runServer(ctx *cli.Context) error {
	// config file
	configFilePath := ctx.String("config")
	if err := config.InitCfg(configFilePath); err != nil {
		return err
	}

	// config file watcher
	watcher, err := config.AddCfgFileWatcher(configFilePath)
	if err != nil {
		return err
	}
	// ============= service start =============

	// db
	dbDao, err := dao.NewGormDBNotAutoMigrate(config.Cfg.DB.Mysql)
	if err != nil {
		return fmt.Errorf("dao.NewGormDB err: %s", err.Error())
	}

	// das core
	dasCore, _, txBuilderBase, err := config.InitDasCore(ctxServer, &wgServer)
	if err != nil {
		return fmt.Errorf("config.InitDasCore err: %s", err.Error())
	}

	// tool refund
	toolRefund := refund.ToolRefund{
		Ctx:           ctxServer,
		Wg:            &wgServer,
		DbDao:         dbDao,
		DasCore:       dasCore,
		TxBuilderBase: txBuilderBase,
	}
	if err := toolRefund.InitRefundInfo(); err != nil {
		return fmt.Errorf("InitRefundInfo err: %s", err.Error())
	}
	toolRefund.RunRefundOnce()
	if err := toolRefund.RunRefund(); err != nil {
		return fmt.Errorf("RunRefund err: %s", err.Error())
	}

	// ============= service end =============
	toolib.ExitMonitoring(func(sig os.Signal) {
		log.Warn("ExitMonitoring:", sig.String())
		if watcher != nil {
			log.Warn("close watcher ... ")
			_ = watcher.Close()
		}
		cancel()
		wgServer.Wait()
		log.Warn("success exit server. bye bye!")
		time.Sleep(time.Second)
		exit <- struct{}{}
	})

	<-exit
	return nil
}
