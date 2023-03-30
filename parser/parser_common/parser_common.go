package parser_common

import (
	"context"
	"fmt"
	"github.com/dotbitHQ/unipay/config"
	"github.com/dotbitHQ/unipay/dao"
	"github.com/dotbitHQ/unipay/notify"
	"github.com/dotbitHQ/unipay/tables"
	"github.com/scorpiotzh/mylog"
	"sync"
	"sync/atomic"
	"time"
)

var log = mylog.NewLogger("parser_common", mylog.LevelDebug)

type ParserApi interface {
	GetLatestBlockNumber() (uint64, error)
	Init(*ParserCore) error
	SingleParsing(*ParserCore) error
	ConcurrentParsing(*ParserCore) error
}

type ParserCore struct {
	Ctx                context.Context
	Wg                 *sync.WaitGroup
	DbDao              *dao.DbDao
	ParserType         tables.ParserType
	PayTokenId         tables.PayTokenId
	Address            string
	CurrentBlockNumber uint64
	ConcurrencyNum     uint64
	ConfirmNum         uint64
	Switch             bool
}

func (p *ParserCore) CheckFork(parentHash string) (bool, error) {
	block, err := p.DbDao.FindBlockInfoByBlockNumber(p.ParserType, p.CurrentBlockNumber-1)
	if err != nil {
		return false, err
	}
	if block.Id > 0 && block.BlockHash != parentHash {
		log.Warn("CheckFork:", p.CurrentBlockNumber, parentHash, block.BlockHash)
		return true, nil
	}
	return false, nil
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
				log.Info("ConcurrentParsing:", concurrencyNum, p.PC.CurrentBlockNumber, latestBlockNumber, confirmNum, concurrencyNum)
				nowTime := time.Now()
				if err := p.PA.ConcurrentParsing(p.PC); err != nil {
					log.Error("ConcurrentParsing err:", parserType, err.Error(), p.PC.CurrentBlockNumber)
					notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, fmt.Sprintf("Parser %d", parserType), err.Error())
				}
				log.Warn("ConcurrentParsing time:", parserType, time.Since(nowTime).Seconds())
				time.Sleep(time.Second * 1)
			} else if p.PC.CurrentBlockNumber < (latestBlockNumber - confirmNum) {
				nowTime := time.Now()
				if err := p.PA.SingleParsing(p.PC); err != nil {
					log.Error("SingleParsing err:", parserType, err.Error(), p.PC.CurrentBlockNumber)
					notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, fmt.Sprintf("Parser %d", parserType), err.Error())
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
