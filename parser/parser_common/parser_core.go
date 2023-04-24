package parser_common

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unipay/dao"
	"unipay/notify"
	"unipay/tables"
)

type ParserCore struct {
	Ctx                context.Context
	Wg                 *sync.WaitGroup
	DbDao              *dao.DbDao
	CN                 *notify.CallbackNotice
	ParserType         tables.ParserType
	PayTokenId         tables.PayTokenId
	Address            string
	ContractAddress    string
	ContractPayTokenId tables.PayTokenId
	CurrentBlockNumber uint64
	ConcurrencyNum     uint64
	ConfirmNum         uint64
	Switch             bool
}

func (p *ParserCore) HandleFork(blockHash, parentHash string) (bool, error) {
	block, err := p.DbDao.FindBlockInfoByBlockNumber(p.ParserType, p.CurrentBlockNumber-1)
	if err != nil {
		return false, err
	}
	if block.Id > 0 && block.BlockHash != parentHash {
		log.Warn("DoCheckFork is true:", p.ParserType, p.CurrentBlockNumber, blockHash, parentHash, block.BlockHash)
		if err := p.DbDao.DeleteBlockInfoByBlockNumber(p.ParserType, p.CurrentBlockNumber-1); err != nil {
			return false, fmt.Errorf("DeleteBlockInfoByBlockNumber err: %s", err.Error())
		}
		atomic.AddUint64(&p.CurrentBlockNumber, ^uint64(0))
		return true, nil
	}
	return false, nil
}

func (p *ParserCore) HandleSingleParsingOK(blockHash, parentHash string) error {
	blockInfo := tables.TableBlockParserInfo{
		ParserType:  p.ParserType,
		BlockNumber: p.CurrentBlockNumber,
		BlockHash:   blockHash,
		ParentHash:  parentHash,
	}
	if err := p.DbDao.CreateBlockInfo(blockInfo); err != nil {
		return fmt.Errorf("CreateBlockInfo err: %s", err.Error())
	} else {
		atomic.AddUint64(&p.CurrentBlockNumber, 1)
	}
	if err := p.DbDao.DeleteBlockInfo(p.ParserType, p.CurrentBlockNumber-20); err != nil {
		log.Error("DeleteBlockInfo1 err:", p.ParserType, err.Error(), p.CurrentBlockNumber)
	}
	return nil
}

func (p *ParserCore) HandleConcurrentParsingOK(blockList []tables.TableBlockParserInfo) error {
	if err := p.DbDao.CreateBlockInfoList(blockList); err != nil {
		return fmt.Errorf("CreateBlockInfoList err:%s", err.Error())
	} else {
		atomic.AddUint64(&p.CurrentBlockNumber, p.ConcurrencyNum)
	}
	if err := p.DbDao.DeleteBlockInfo(p.ParserType, p.CurrentBlockNumber-20); err != nil {
		log.Error("DeleteBlockInfo2 err:", p.ParserType, err.Error(), p.CurrentBlockNumber)
	}
	return nil
}

func (p *ParserCore) HandlePayment(paymentInfo tables.TablePaymentInfo, orderInfo tables.TableOrderInfo) error {
	noticeInfo := tables.TableNoticeInfo{
		OrderId:      paymentInfo.OrderId,
		EventType:    tables.EventTypeOrderPay,
		NoticeCount:  0,
		NoticeStatus: tables.NoticeStatusDefault,
		Timestamp:    time.Now().UnixMilli(),
	}

	orderInfo.PayStatus = tables.PayStatusPaid
	if err := p.CN.CallbackNotice(noticeInfo, paymentInfo, orderInfo); err != nil {
		log.Error("CallbackNotice err: %s", err.Error())
	} else {
		noticeInfo.NoticeStatus = tables.NoticeStatusOK
	}

	if err := p.DbDao.UpdatePaymentStatus(paymentInfo, noticeInfo); err != nil {
		return fmt.Errorf("UpdatePaymentStatus err: %s", err.Error())
	}
	return nil
}
