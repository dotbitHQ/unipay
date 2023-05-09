package parser_common

import (
	"fmt"
	"github.com/scorpiotzh/mylog"
	"strings"
	"sync/atomic"
	"time"
	"unipay/config"
	"unipay/notify"
)

var log = mylog.NewLogger("parser_common", mylog.LevelDebug)

type ParserApi interface {
	GetLatestBlockNumber() (uint64, error)
	Init(*ParserCore) error
	SingleParsing(*ParserCore) error
	ConcurrentParsing(*ParserCore) error
}

type ParserCommon struct {
	PC *ParserCore
	PA ParserApi
}

func (p *ParserCommon) initCurrentBlockNumber() error {
	if block, err := p.PC.DbDao.FindBlockInfo(p.PC.ParserType); err != nil {
		return err
	} else if block.Id > 0 {
		p.PC.CurrentBlockNumber = block.BlockNumber
	} else {
		currentBlockNumber, err := p.PA.GetLatestBlockNumber()
		if err != nil {
			return fmt.Errorf("handle err: %s", err.Error())
		}
		p.PC.CurrentBlockNumber = currentBlockNumber
	}
	return nil
}

func (p *ParserCommon) Parser() {
	if err := p.PA.Init(p.PC); err != nil {
		log.Error("Parser Init err: %s", err.Error())
		return
	}
	if err := p.initCurrentBlockNumber(); err != nil {
		log.Error("initCurrentBlockNumber err: ", err.Error())
		return
	}
	parserType := p.PC.ParserType
	concurrencyNum := p.PC.ConcurrencyNum
	confirmNum := p.PC.ConfirmNum

	atomic.AddUint64(&p.PC.CurrentBlockNumber, 1)
	p.PC.Wg.Add(1)
	for {
		select {
		default:
			latestBlockNumber, err := p.PA.GetLatestBlockNumber()
			if err != nil {
				log.Error("GetLatestBlockNumber err: ", err.Error())
				time.Sleep(time.Second * 10)
			} else if concurrencyNum > 1 && p.PC.CurrentBlockNumber < (latestBlockNumber-confirmNum-concurrencyNum) {
				log.Info("ConcurrentParsing:", p.PC.CurrentBlockNumber, latestBlockNumber)
				nowTime := time.Now()
				if err := p.PA.ConcurrentParsing(p.PC); err != nil {
					log.Error("ConcurrentParsing err:", parserType, err.Error(), p.PC.CurrentBlockNumber)
					if !strings.Contains(err.Error(), "data is nil") {
						notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, fmt.Sprintf("Parser %d", parserType), err.Error())
					}
				}
				log.Warn("ConcurrentParsing time:", parserType, time.Since(nowTime).Seconds())
				time.Sleep(time.Second * 1)
			} else if p.PC.CurrentBlockNumber < (latestBlockNumber - confirmNum) {
				nowTime := time.Now()
				if err := p.PA.SingleParsing(p.PC); err != nil {
					log.Error("SingleParsing err:", parserType, err.Error(), p.PC.CurrentBlockNumber)
					if !strings.Contains(err.Error(), "data is nil") {
						notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, fmt.Sprintf("Parser %d", parserType), err.Error())
					}
				}
				log.Warn("Parsing time:", parserType, time.Since(nowTime).Seconds())
				time.Sleep(time.Second * 5)
			} else {
				log.Info("Parser:", parserType, p.PC.CurrentBlockNumber, latestBlockNumber)
				time.Sleep(time.Second * 10)
			}
		case <-p.PC.Ctx.Done():
			log.Warn("Parser done", parserType)
			p.PC.Wg.Done()
			return
		}
	}
}
