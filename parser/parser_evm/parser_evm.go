package parser_evm

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/chain/chain_evm"
	"github.com/dotbitHQ/unipay/parser/parser_common"
	"github.com/dotbitHQ/unipay/tables"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/scorpiotzh/mylog"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var log = mylog.NewLogger("parser_evm", mylog.LevelDebug)

type ParserEvm struct {
	parser_common.ParserCommon
	ChainEvm *chain_evm.ChainEvm
}

func (p *ParserEvm) getLatestBlockNumber() (uint64, error) {
	currentBlockNumber, err := p.ChainEvm.BestBlockNumber()
	if err != nil {
		log.Error("BestBlockNumber err: ", p.ParserType, err.Error())
		return 0, fmt.Errorf("BestBlockNumber err: %s", err.Error())
	}
	return currentBlockNumber, nil
}

func (p *ParserEvm) Parser() {
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
				log.Warn("parserConcurrencyMode time:", p.ParserType, time.Since(nowTime).Seconds())
				time.Sleep(time.Second * 1)
			} else if p.CurrentBlockNumber < (latestBlockNumber - p.ConfirmNum) {
				nowTime := time.Now()
				if err := p.parserSubMode(); err != nil {
					log.Error("parserSubMode err:", p.ParserType, err.Error(), p.CurrentBlockNumber)
				}
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

func (p *ParserEvm) parsingBlockData(block *chain_evm.Block) error {
	for _, tx := range block.Transactions {
		switch strings.ToLower(ethcommon.HexToAddress(tx.To).Hex()) {
		case strings.ToLower(p.Address):
			orderId := string(ethcommon.FromHex(tx.Input))
			log.Info("ParsingBlockData:", p.ParserType, tx.Hash, tx.From, orderId, tx.Value)
			if orderId == "" {
				continue
			}
			// select order by order id which in tx memo
			order, err := p.DbDao.GetOrderInfoByOrderId(orderId)
			if err != nil {
				return fmt.Errorf("GetOrderInfoByOrderId err: %s", err.Error())
			} else if order.Id == 0 {
				log.Warn("order not exist:", p.ParserType, orderId)
				continue
			}
			if order.PayTokenId != p.PayTokenId {
				log.Warn("order pay token id not match", order.OrderId, p.PayTokenId)
				continue
			}
			// check value is equal amount or not
			decValue := decimal.NewFromBigInt(chain_evm.BigIntFromHex(tx.Value), 0)
			if decValue.Cmp(order.Amount) == -1 {
				log.Warn("tx value less than order amount:", p.ParserType, decValue, order.Amount.String())
				continue
			}
			// change the status to confirm
			timestamp, _ := strconv.ParseInt(block.Timestamp, 10, 64)
			paymentInfo := tables.TablePaymentInfo{
				Id:            0,
				PayHash:       tx.Hash,
				OrderId:       order.OrderId,
				PayAddress:    ethcommon.HexToAddress(tx.From).Hex(),
				AlgorithmId:   order.AlgorithmId,
				Timestamp:     timestamp,
				Amount:        order.Amount,
				PayHashStatus: tables.PayHashStatusConfirm,
				RefundStatus:  tables.RefundStatusDefault,
				RefundHash:    "",
				RefundNonce:   0,
			}
			if err := p.DbDao.UpdatePaymentStatus(paymentInfo); err != nil {
				return fmt.Errorf("UpdatePaymentStatus err: %s", err.Error())
			}
		}
		continue
	}
	return nil
}

func (p *ParserEvm) parserSubMode() error {
	log.Info("parserSubMode:", p.ParserType, p.CurrentBlockNumber)

	block, err := p.ChainEvm.GetBlockByNumber(p.CurrentBlockNumber)
	if err != nil {
		return fmt.Errorf("GetBlockByNumber err: %s", err.Error())
	}
	blockHash := block.Hash
	parentHash := block.ParentHash
	if block.Hash == "" || block.ParentHash == "" {
		log.Info("GetBlockByNumber:", p.CurrentBlockNumber, toolib.JsonString(&block))
		return fmt.Errorf("GetBlockByNumber data is nil: [%d]", p.CurrentBlockNumber)
	}
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

func (p *ParserEvm) parserConcurrencyMode() error {
	log.Info("parserConcurrencyMode:", p.ParserType, p.CurrentBlockNumber, p.ConcurrencyNum)

	var blockList = make([]tables.TableBlockParserInfo, p.ConcurrencyNum)
	var blocks = make([]*chain_evm.Block, p.ConcurrencyNum)
	var blockCh = make(chan *chain_evm.Block, p.ConcurrencyNum)

	blockLock := &sync.Mutex{}
	blockGroup := &errgroup.Group{}

	for i := uint64(0); i < p.ConcurrencyNum; i++ {
		bn := p.CurrentBlockNumber + i
		index := i
		blockGroup.Go(func() error {
			block, err := p.ChainEvm.GetBlockByNumber(bn)
			if err != nil {
				return fmt.Errorf("GetBlockByNumber err:%s [%d]", err.Error(), bn)
			}
			if block.Hash == "" || block.ParentHash == "" {
				log.Warn("GetBlockByNumber:", bn, toolib.JsonString(&block))
				return fmt.Errorf("GetBlockByNumber data is nil: [%d]", bn)
			}
			hash := block.Hash
			parentHash := block.ParentHash

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
