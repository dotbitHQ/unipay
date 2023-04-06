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
	"unipay/http_svr"
	"unipay/http_svr/handle"
	"unipay/notify"
	"unipay/parser"
	"unipay/timer"
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
	dbDao, err := dao.NewGormDB(config.Cfg.DB.Mysql)
	if err != nil {
		return fmt.Errorf("dao.NewGormDB err: %s", err.Error())
	}

	// das core
	dasCore, _, _, err := config.InitDasCore(ctxServer, &wgServer)
	if err != nil {
		return fmt.Errorf("config.InitDasCore err: %s", err.Error())
	}

	// http
	httpSvr := http_svr.HttpSvr{
		Ctx:     ctxServer,
		Address: config.Cfg.Server.HttpPort,
		H: &handle.HttpHandle{
			Ctx:     ctxServer,
			DbDao:   dbDao,
			DasCore: dasCore,
		},
	}
	httpSvr.Run()

	// callback notice
	cn := &notify.CallbackNotice{DbDao: dbDao}

	// tool parser
	toolParser, err := parser.NewToolParser(ctxServer, &wgServer, dbDao, cn)
	if err != nil {
		return fmt.Errorf("NewToolParser err: %s", err.Error())
	}
	toolParser.RunParser()

	// tool timer
	toolTimer := timer.ToolTimer{
		Ctx:   ctxServer,
		Wg:    &wgServer,
		DbDao: dbDao,
		CN:    cn,
	}
	toolTimer.RunCallbackNotice()

	// ============= service end =============
	toolib.ExitMonitoring(func(sig os.Signal) {
		log.Warn("ExitMonitoring:", sig.String())
		if watcher != nil {
			log.Warn("close watcher ... ")
			_ = watcher.Close()
		}
		httpSvr.Shutdown()
		cancel()
		wgServer.Wait()
		log.Warn("success exit server. bye bye!")
		time.Sleep(time.Second)
		exit <- struct{}{}
	})

	<-exit
	return nil
}
