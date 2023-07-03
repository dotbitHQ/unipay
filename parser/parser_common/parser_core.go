package parser_common

import (
	"context"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/shopspring/decimal"
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
	ContractAddress    string
	ContractPayTokenId tables.PayTokenId
	CurrentBlockNumber uint64
	ConcurrencyNum     uint64
	ConfirmNum         uint64
	Switch             bool
	AddrMap            map[string]string
}

func (p *ParserCore) CreatePaymentForMismatch(algorithmId common.DasAlgorithmId, orderId, payHash, payAddress string, amount decimal.Decimal, payTokenId tables.PayTokenId) {
	paymentInfo := tables.TablePaymentInfo{
		PayHash:       payHash,
		OrderId:       orderId,
		PayAddress:    payAddress,
		AlgorithmId:   algorithmId,
		Timestamp:     time.Now().UnixMilli(),
		Amount:        amount,
		PayTokenId:    payTokenId,
		PayHashStatus: tables.PayHashStatusConfirm,
		RefundStatus:  tables.RefundStatusDefault,
	}
	if err := p.DbDao.CreatePayment(paymentInfo); err != nil {
		log.Error("CreatePaymentForAmountMismatch err:", orderId, payHash, err.Error())
	}
}

func (p *ParserCore) CreatePaymentForAmountMismatch(order tables.TableOrderInfo, payHash, payAddress string, amount decimal.Decimal) {
	paymentInfo := tables.TablePaymentInfo{
		PayHash:       payHash,
		OrderId:       order.OrderId,
		PayAddress:    payAddress,
		AlgorithmId:   order.AlgorithmId,
		Timestamp:     time.Now().UnixMilli(),
		Amount:        amount,
		PayTokenId:    order.PayTokenId,
		PayHashStatus: tables.PayHashStatusConfirm,
		RefundStatus:  tables.RefundStatusDefault,
	}
	if err := p.DbDao.CreatePayment(paymentInfo); err != nil {
		log.Error("CreatePaymentForAmountMismatch err:", order.OrderId, payHash, err.Error())
	}
}

func (p *ParserCore) DoPayment(order tables.TableOrderInfo, txId, fromHex string) error {
	paymentInfo := tables.TablePaymentInfo{
		PayHash:       txId,
		OrderId:       order.OrderId,
		PayAddress:    fromHex,
		AlgorithmId:   order.AlgorithmId,
		Timestamp:     time.Now().UnixMilli(),
		Amount:        order.Amount,
		PayTokenId:    order.PayTokenId,
		PayHashStatus: tables.PayHashStatusConfirm,
		RefundStatus:  tables.RefundStatusDefault,
	}
	if err := p.CN.HandlePayment(paymentInfo, order); err != nil {
		return fmt.Errorf("HandlePayment err: %s", err.Error())
	}
	return nil
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
