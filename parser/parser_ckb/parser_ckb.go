package parser_ckb

import (
	"context"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/http_api/logger"
	"github.com/nervosnetwork/ckb-sdk-go/address"
	"github.com/nervosnetwork/ckb-sdk-go/rpc"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
	"strconv"
	"sync"
	"unipay/config"
	"unipay/parser/parser_common"
	"unipay/tables"
)

var log = logger.NewLogger("parser_ckb", logger.LevelDebug)

type ParserCkb struct {
	Ctx    context.Context
	Client rpc.Client
}

func (p *ParserCkb) Init(pc *parser_common.ParserCore) error {
	return nil
}
func (p *ParserCkb) GetLatestBlockNumber() (uint64, error) {
	if blockNumber, err := p.Client.GetTipBlockNumber(p.Ctx); err != nil {
		return 0, fmt.Errorf("GetTipBlockNumber err: %s", err.Error())
	} else {
		return blockNumber, nil
	}
}
func (p *ParserCkb) SingleParsing(pc *parser_common.ParserCore) error {
	parserType, currentBlockNumber := pc.ParserType, pc.CurrentBlockNumber
	log.Debug("SingleParsing:", parserType, currentBlockNumber)

	block, err := p.Client.GetBlockByNumber(p.Ctx, currentBlockNumber)
	if err != nil {
		return fmt.Errorf("GetBlockByNumber err: %s", err.Error())
	}

	blockHash := block.Header.Hash.Hex()
	parentHash := block.Header.ParentHash.Hex()
	log.Debug("SingleParsing:", parserType, blockHash, parentHash)

	if isFork, err := pc.HandleFork(blockHash, parentHash); err != nil {
		return fmt.Errorf("HandleFork err: %s", err.Error())
	} else if isFork {
		return nil
	}

	if err := p.parsingBlockData(block, pc); err != nil {
		return fmt.Errorf("parsingBlockData err: %s", err.Error())
	} else {
		if err := pc.HandleSingleParsingOK(blockHash, parentHash); err != nil {
			return fmt.Errorf("HandleSingleParsingOK err: %s", err.Error())
		}
	}
	return nil
}
func (p *ParserCkb) ConcurrentParsing(pc *parser_common.ParserCore) error {
	parserType, concurrencyNum, currentBlockNumber := pc.ParserType, pc.ConcurrencyNum, pc.CurrentBlockNumber
	log.Debug("ConcurrentParsing:", parserType, concurrencyNum, currentBlockNumber)

	var blockList = make([]tables.TableBlockParserInfo, concurrencyNum)
	var blocks = make([]*types.Block, concurrencyNum)
	var blockCh = make(chan *types.Block, concurrencyNum)

	blockLock := &sync.Mutex{}
	blockGroup := &errgroup.Group{}

	for i := uint64(0); i < concurrencyNum; i++ {
		bn := currentBlockNumber + i
		index := i
		blockGroup.Go(func() error {
			block, err := p.Client.GetBlockByNumber(p.Ctx, bn)
			if err != nil {
				return fmt.Errorf("GetBlockByNumber err:%s [%d]", err.Error(), bn)
			}
			hash := block.Header.Hash.Hex()
			parentHash := block.Header.ParentHash.Hex()

			blockLock.Lock()
			blockList[index] = tables.TableBlockParserInfo{
				ParserType:  parserType,
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
			if err := p.parsingBlockData(v, pc); err != nil {
				return fmt.Errorf("parsingBlockData err: %s", err.Error())
			}
		}
		return nil
	})

	if err := blockGroup.Wait(); err != nil {
		return fmt.Errorf("errGroup.Wait()2 err: %s", err.Error())
	}

	// ok
	if err := pc.HandleConcurrentParsingOK(blockList); err != nil {
		return fmt.Errorf("HandleConcurrentParsingOK err: %s", err.Error())
	}
	return nil
}

func (p *ParserCkb) parsingBlockData(block *types.Block, pc *parser_common.ParserCore) error {
	parserType := pc.ParserType
	if block == nil {
		return fmt.Errorf("block is nil")
	}
	log.Info("parsingBlockData:", toolib.JsonString(pc.AddrMap))
	for _, tx := range block.Transactions {
		for i, v := range tx.Outputs {
			addrArgs := common.Bytes2Hex(v.Lock.Args)
			_, ok := pc.AddrMap[addrArgs]
			if !ok {
				continue
			}
			orderId := string(tx.OutputsData[i])
			if orderId == "" {
				continue
			}
			log.Info("parsingBlockData:", orderId, tx.Hash.Hex())
			txInputs, err := p.Client.GetTransaction(p.Ctx, tx.Inputs[0].PreviousOutput.TxHash)
			if err != nil {
				return fmt.Errorf("GetTransaction err:%s", err.Error())
			}
			mode := address.Mainnet
			if config.Cfg.Server.Net != common.DasNetTypeMainNet {
				mode = address.Testnet
			}
			fromAddr, err := common.ConvertScriptToAddress(mode, txInputs.Transaction.Outputs[tx.Inputs[0].PreviousOutput.Index].Lock)
			if err != nil {
				return fmt.Errorf("common.ConvertScriptToAddress err:%s", err.Error())
			}

			capacity, _ := decimal.NewFromString(strconv.FormatUint(v.Capacity, 10))
			order, err := pc.DbDao.GetOrderInfoByOrderIdWithAddr(orderId, addrArgs)
			if err != nil {
				return fmt.Errorf("GetOrderInfoByOrderIdWithAddr err: %s", err.Error())
			} else if order.Id == 0 {
				log.Warn("order not exist:", parserType, orderId)
				//pc.CreatePaymentForMismatch("", tx.Hash.Hex(), fromAddr, capacity, pc.PayTokenId)
				continue
			}
			if order.PayTokenId != tables.PayTokenIdCKB &&
				order.PayTokenId != tables.PayTokenIdDAS &&
				order.PayTokenId != tables.PayTokenIdCkbCCC {
				log.Warn("order pay token id not match", order.OrderId)
				//pc.CreatePaymentForMismatch(common.DasAlgorithmIdCkb, order.OrderId, tx.Hash.Hex(), fromAddr, capacity, tables.PayTokenIdCKB)
				continue
			}
			if capacity.Cmp(order.Amount) == -1 {
				log.Warn("tx value less than order amount:", capacity.String(), order.Amount.String())
				pc.CreatePaymentForMismatch(order.OrderId, tx.Hash.Hex(), fromAddr, capacity, order.PayTokenId)
				continue
			}
			// change the status to confirm
			if err = pc.DoPayment(order, tx.Hash.Hex(), fromAddr, pc.ParserType.ToAlgorithmId()); err != nil {
				return fmt.Errorf("pc.DoPayment err: %s", err.Error())
			}
			break
		}
	}
	return nil
}
