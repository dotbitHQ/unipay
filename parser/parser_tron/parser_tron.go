package parser_tron

import (
	"encoding/hex"
	"fmt"
	"github.com/dotbitHQ/das-lib/chain/chain_tron"
	"github.com/dotbitHQ/unipay/config"
	"github.com/dotbitHQ/unipay/notify"
	"github.com/dotbitHQ/unipay/parser/parser_common"
	"github.com/dotbitHQ/unipay/tables"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/golang/protobuf/proto"
	"github.com/scorpiotzh/mylog"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
	"sync"
	"sync/atomic"
	"time"
)

var log = mylog.NewLogger("parser_tron", mylog.LevelDebug)

type ParserTron struct {
	parser_common.ParserCommon
	ChainTron *chain_tron.ChainTron
}

func (p *ParserTron) getLatestBlockNumber() (uint64, error) {
	currentBlockNumber, err := p.ChainTron.GetBlockNumber()
	if err != nil {
		return 0, fmt.Errorf("GetBlockNumber err: %s", err.Error())
	}
	return uint64(currentBlockNumber), nil
}

func (p *ParserTron) Parser() {
	if err := p.InitCurrentBlockNumber(p.getLatestBlockNumber); err != nil {
		log.Error("InitCurrentBlockNumber err: ", err.Error())
		return
	}
	atomic.AddUint64(&p.CurrentBlockNumber, 1)
	p.Wg.Add(1)
	for {
		select {
		default:
			latestBlockNumber, err := p.getLatestBlockNumber()
			if err != nil {
				log.Error("getLatestBlockNumber err: ", err.Error())
				time.Sleep(time.Second * 10)
			} else if p.ConcurrencyNum > 1 && p.CurrentBlockNumber < (latestBlockNumber-p.ConfirmNum-p.ConcurrencyNum) {
				nowTime := time.Now()
				if err := p.parserConcurrencyMode(); err != nil {
					log.Error("parserConcurrencyMode err:", p.ParserType, err.Error(), p.CurrentBlockNumber)
				}
				notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, fmt.Sprintf("ParserType %d", p.ParserType), err.Error())
				log.Warn("parserConcurrencyMode time:", p.ParserType, time.Since(nowTime).Seconds())
				time.Sleep(time.Second * 1)
			} else if p.CurrentBlockNumber < (latestBlockNumber - p.ConfirmNum) {
				nowTime := time.Now()
				if err := p.parserSubMode(); err != nil {
					log.Error("parserSubMode err:", p.ParserType, err.Error(), p.CurrentBlockNumber)
				}
				notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, fmt.Sprintf("ParserType %d", p.ParserType), err.Error())
				log.Warn("parserSubMode time:", p.ParserType, time.Since(nowTime).Seconds())
				time.Sleep(time.Second * 5)
			} else {
				log.Info("Parser:", p.ParserType, p.CurrentBlockNumber, latestBlockNumber)
				time.Sleep(time.Second * 10)
			}
		case <-p.Ctx.Done():
			log.Warn("Parser done", p.ParserType)
			p.Wg.Done()
			return
		}
	}
}

func (p *ParserTron) parsingBlockData(block *api.BlockExtention) error {
	if block == nil {
		return fmt.Errorf("block is nil")
	}
	for _, tx := range block.Transactions {
		if len(tx.Transaction.RawData.Contract) != 1 {
			continue
		}
		orderId := chain_tron.GetMemo(tx.Transaction.RawData.Data)
		if orderId == "" {
			continue
		} else if len(orderId) > 64 {
			continue
		}

		switch tx.Transaction.RawData.Contract[0].Type {
		case core.Transaction_Contract_TransferContract:
			instance := core.TransferContract{}
			if err := proto.Unmarshal(tx.Transaction.RawData.Contract[0].Parameter.Value, &instance); err != nil {
				log.Error(" proto.Unmarshal err:", err.Error())
				continue
			}
			fromAddr, toAddr := hex.EncodeToString(instance.OwnerAddress), hex.EncodeToString(instance.ToAddress)
			if toAddr != p.Address {
				continue
			}
			log.Info("parsingBlockData orderId:", orderId, hex.EncodeToString(tx.Txid))

			// check order id
			order, err := p.DbDao.GetOrderInfoByOrderId(orderId)
			if err != nil {
				return fmt.Errorf("GetOrderInfoByOrderId err: %s", err.Error())
			} else if order.Id == 0 {
				log.Warn("GetOrderInfoByOrderId is not exist:", p.ParserType, orderId)
				continue
			}
			if order.PayTokenId != p.PayTokenId {
				log.Warn("order pay token id not match", order.OrderId)
				continue
			}

			amountValue := decimal.New(instance.Amount, 0)
			if amountValue.Cmp(order.Amount) == -1 {
				log.Warn("tx value less than order amount:", amountValue.String(), order.Amount.String())
				continue
			}
			// change the status to confirm
			paymentInfo := tables.TablePaymentInfo{
				Id:            0,
				PayHash:       hex.EncodeToString(tx.Txid),
				OrderId:       order.OrderId,
				PayAddress:    fromAddr,
				AlgorithmId:   order.AlgorithmId,
				Timestamp:     tx.Transaction.RawData.Timestamp,
				Amount:        order.Amount,
				PayHashStatus: tables.PayHashStatusConfirm,
				RefundStatus:  tables.RefundStatusDefault,
				RefundHash:    "",
				RefundNonce:   0,
			}
			if err := p.DbDao.UpdatePaymentStatus(paymentInfo); err != nil {
				return fmt.Errorf("UpdatePaymentStatus err: %s", err.Error())
			}
		case core.Transaction_Contract_TransferAssetContract:
		case core.Transaction_Contract_TriggerSmartContract:
		}
	}
	return nil
}

func (p *ParserTron) parserSubMode() error {
	log.Info("parserSubMode:", p.ParserType, p.CurrentBlockNumber)

	block, err := p.ChainTron.GetBlockByNumber(p.CurrentBlockNumber)
	if err != nil {
		return fmt.Errorf("GetBlockByNumber err: %s", err.Error())
	}
	if block.BlockHeader == nil {
		return fmt.Errorf("parserSubMode: block.BlockHeader is nil[%d]", p.CurrentBlockNumber)
	} else if block.BlockHeader.RawData == nil {
		return fmt.Errorf("parserSubMode: block.BlockHeader.RawData is nil[%d]", p.CurrentBlockNumber)
	}

	blockHash := hex.EncodeToString(block.Blockid)
	parentHash := hex.EncodeToString(block.BlockHeader.RawData.ParentHash)
	log.Info("parserSubMode:", p.ParserType, blockHash, parentHash)

	if fork, err := p.CheckFork(parentHash); err != nil {
		return fmt.Errorf("CheckFork err: %s", err.Error())
	} else if fork {
		log.Warn("CheckFork is true:", p.ParserType, p.CurrentBlockNumber, blockHash, parentHash)
		if err := p.DbDao.DeleteBlockInfoByBlockNumber(p.ParserType, p.CurrentBlockNumber-1); err != nil {
			return fmt.Errorf("DeleteBlockInfoByBlockNumber err: %s", err.Error())
		}
		atomic.AddUint64(&p.CurrentBlockNumber, ^uint64(0))
	} else if err := p.parsingBlockData(block); err != nil {
		return fmt.Errorf("parsingBlockData err: %s", err.Error())
	} else {
		blockInfo := tables.TableBlockParserInfo{
			ParserType:  p.ParserType,
			BlockNumber: p.CurrentBlockNumber,
			BlockHash:   blockHash,
			ParentHash:  parentHash,
		}
		if err = p.DbDao.CreateBlockInfo(blockInfo); err != nil {
			return fmt.Errorf("CreateBlockInfo err: %s", err.Error())
		} else {
			atomic.AddUint64(&p.CurrentBlockNumber, 1)
		}
		if err = p.DbDao.DeleteBlockInfo(p.ParserType, p.CurrentBlockNumber-20); err != nil {
			return fmt.Errorf("DeleteBlockInfo err: %s", err.Error())
		}
	}
	return nil
}

func (p *ParserTron) parserConcurrencyMode() error {
	log.Info("parserConcurrencyMode:", p.ParserType, p.CurrentBlockNumber, p.ConcurrencyNum)

	var blockList = make([]tables.TableBlockParserInfo, p.ConcurrencyNum)
	var blocks = make([]*api.BlockExtention, p.ConcurrencyNum)
	var blockCh = make(chan *api.BlockExtention, p.ConcurrencyNum)

	blockLock := &sync.Mutex{}
	blockGroup := &errgroup.Group{}

	for i := uint64(0); i < p.ConcurrencyNum; i++ {
		bn := p.CurrentBlockNumber + i
		index := i
		blockGroup.Go(func() error {
			block, err := p.ChainTron.GetBlockByNumber(bn)
			if err != nil {
				return fmt.Errorf("GetBlockByNumber err:%s [%d]", err.Error(), bn)
			}
			if block.BlockHeader == nil {
				return fmt.Errorf("parserSubMode: block.BlockHeader is nil[%d]", bn)
			} else if block.BlockHeader.RawData == nil {
				return fmt.Errorf("parserSubMode: block.BlockHeader.RawData is nil[%d]", bn)
			}

			hash := hex.EncodeToString(block.Blockid)
			parentHash := hex.EncodeToString(block.BlockHeader.RawData.ParentHash)

			blockLock.Lock()
			blockList[index] = tables.TableBlockParserInfo{
				ParserType:  p.ParserType,
				BlockNumber: bn,
				BlockHash:   hash,
				ParentHash:  parentHash,
			}
			blocks[index] = block
			blockLock.Unlock()

			return nil
		})
	}
	if err := blockGroup.Wait(); err != nil {
		return fmt.Errorf("errGroup.Wait()1 err: %s", err.Error())
	}

	for i := range blocks {
		blockCh <- blocks[i]
	}
	close(blockCh)

	blockGroup.Go(func() error {
		for v := range blockCh {
			if err := p.parsingBlockData(v); err != nil {
				return fmt.Errorf("parsingBlockData err: %s", err.Error())
			}
		}
		return nil
	})

	if err := blockGroup.Wait(); err != nil {
		return fmt.Errorf("errGroup.Wait()2 err: %s", err.Error())
	}

	// block
	if err := p.DbDao.CreateBlockInfoList(blockList); err != nil {
		return fmt.Errorf("CreateBlockInfoList err:%s", err.Error())
	} else {
		atomic.AddUint64(&p.CurrentBlockNumber, p.ConcurrencyNum)
	}
	if err := p.DbDao.DeleteBlockInfo(p.ParserType, p.CurrentBlockNumber-20); err != nil {
		log.Error("DeleteBlockInfo err:", p.ParserType, err.Error())
	}
	return nil
}
