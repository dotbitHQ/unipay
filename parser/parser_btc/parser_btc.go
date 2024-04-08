package parser_btc

import (
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/dotbitHQ/das-lib/http_api/logger"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
	"sync"
	"unipay/config"
	"unipay/notify"
	"unipay/parser/parser_common"
	"unipay/tables"
)

var log = logger.NewLogger("parser_btc", logger.LevelDebug)

type ParserBtc struct {
	NodeRpc   *rpcclient.Client
	NetParams chaincfg.Params
}

func (p *ParserBtc) GetLatestBlockNumber() (uint64, error) {
	blockNumber, err := p.NodeRpc.GetBlockCount()
	if err != nil {
		return 0, fmt.Errorf("GetBlockCount err: %s", err.Error())
	}
	return uint64(blockNumber), nil
}
func (p *ParserBtc) Init(pc *parser_common.ParserCore) error {
	return nil
}

func (p *ParserBtc) SingleParsing(pc *parser_common.ParserCore) error {
	parserType, currentBlockNumber := pc.ParserType, pc.CurrentBlockNumber
	log.Debug("SingleParsing:", parserType, currentBlockNumber)

	hash, err := p.NodeRpc.GetBlockHash(int64(currentBlockNumber))
	if err != nil {
		return fmt.Errorf("req GetBlockHash err: %s", err.Error())
	}

	block, err := p.NodeRpc.GetBlock(hash)
	if err != nil {
		return fmt.Errorf("req GetBlock err: %s", err.Error())
	}

	blockHash := block.BlockHash().String()
	parentHash := block.Header.PrevBlock.String()
	log.Debug("SingleParsing:", parserType, blockHash, parentHash)

	if isFork, err := pc.HandleFork(blockHash, parentHash); err != nil {
		return fmt.Errorf("HandleFork err: %s", err.Error())
	} else if isFork {
		return nil
	}
	if err := p.parsingBlockData(block, pc, currentBlockNumber); err != nil {
		return fmt.Errorf("parsingBlockData err: %s", err.Error())
	} else {
		if err := pc.HandleSingleParsingOK(blockHash, parentHash); err != nil {
			return fmt.Errorf("HandleSingleParsingOK err: %s", err.Error())
		}
	}
	return nil
}

type blockWithNumber struct {
	block       *wire.MsgBlock
	blockNumber uint64
}

func (p *ParserBtc) ConcurrentParsing(pc *parser_common.ParserCore) error {
	parserType, concurrencyNum, currentBlockNumber := pc.ParserType, pc.ConcurrencyNum, pc.CurrentBlockNumber
	log.Debug("ConcurrentParsing:", parserType, concurrencyNum, currentBlockNumber)

	var blockList = make([]tables.TableBlockParserInfo, concurrencyNum)
	var blocks = make([]blockWithNumber, concurrencyNum)
	var blockCh = make(chan blockWithNumber, concurrencyNum)

	blockLock := &sync.Mutex{}
	blockGroup := &errgroup.Group{}

	for i := uint64(0); i < concurrencyNum; i++ {
		bn := currentBlockNumber + i
		index := i
		blockGroup.Go(func() error {
			blockHash, err := p.NodeRpc.GetBlockHash(int64(bn))
			if err != nil {
				return fmt.Errorf("req GetBlockHash err: %s", err.Error())
			}

			block, err := p.NodeRpc.GetBlock(blockHash)
			if err != nil {
				return fmt.Errorf("req GetBlock err: %s", err.Error())
			}

			hash := block.BlockHash().String()
			parentHash := block.Header.PrevBlock.String()

			blockLock.Lock()
			blockList[index] = tables.TableBlockParserInfo{
				ParserType:  parserType,
				BlockNumber: bn,
				BlockHash:   hash,
				ParentHash:  parentHash,
			}
			blocks[index] = blockWithNumber{block, bn}
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
			if err := p.parsingBlockData(v.block, pc, v.blockNumber); err != nil {
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

func (p *ParserBtc) parsingBlockData(block *wire.MsgBlock, pc *parser_common.ParserCore, blockNumber uint64) error {
	parserType := pc.ParserType
	if block == nil {
		return fmt.Errorf("block is nil")
	}
	log.Debug("parsingBlockData:", parserType, blockNumber, block.BlockHash().String(), len(block.Transactions))

	for _, tx := range block.Transactions {
		// check address of outputs
		_, addrList, _, err := txscript.ExtractPkScriptAddrs(tx.TxOut[0].PkScript, &p.NetParams)
		if err != nil {
			return fmt.Errorf("txscript.ExtractPkScriptAddrs err: %s", err.Error())
		}
		if len(addrList) == 0 {
			continue
		}
		if _, ok := pc.AddrMap[addrList[0].String()]; ok {
			decValue := decimal.NewFromInt(tx.TxOut[0].Value)
			// check inputs & pay info & order id
			pkScript, err := txscript.ComputePkScript(tx.TxIn[0].SignatureScript, tx.TxIn[0].Witness)
			if err != nil {
				return fmt.Errorf("txscript.ComputePkScript err: %s", err.Error())
			}
			addr, err := pkScript.Address(&p.NetParams)
			if err != nil {
				return fmt.Errorf("pkScript.Address err: %s", err.Error())
			}
			log.Info("parsingBlockData:", parserType, tx.TxHash().String(), addr.String(), decValue.String())
			if ok, err := p.dealWithOpReturn(pc, tx, decValue, addr.String(), addrList[0].String()); err != nil {
				return fmt.Errorf("dealWithOpReturn err: %s", err.Error())
			} else if ok {
				continue
			}
			if err = p.dealWithHashAndAmount(pc, tx, decValue, addr.String(), addrList[0].String()); err != nil {
				return fmt.Errorf("dealWithHashAndAmount err: %s", err.Error())
			}
		}
	}
	return nil
}

func (p *ParserBtc) dealWithOpReturn(pc *parser_common.ParserCore, tx *wire.MsgTx, decValue decimal.Decimal, fromAddr, toAddr string) (bool, error) {
	var orderId string
	txOut := tx.TxOut[len(tx.TxOut)-1]
	if txscript.IsNullData(txOut.PkScript) && len(txOut.PkScript) > 32 {
		orderId = hex.EncodeToString(txOut.PkScript[len(txOut.PkScript)-32:])
	}
	log.Info("dealWithOpReturn:", orderId, fromAddr)

	if orderId == "" {
		return false, nil
	}

	order, err := pc.DbDao.GetOrderInfoByOrderIdWithAddr(orderId, toAddr)
	if err != nil {
		return false, fmt.Errorf("GetOrderInfoByOrderIdWithAddr err: %s", err.Error())
	} else if order.Id == 0 {
		log.Warn("order not exist:", pc.ParserType, orderId)
		return false, nil
	}
	if order.PayTokenId != pc.PayTokenId {
		log.Warn("order pay token id not match", order.OrderId)
		return false, nil
	}
	if decValue.Cmp(order.Amount) == -1 {
		log.Warn("tx value less than order amount:", decValue.String(), order.Amount.String())
		pc.CreatePaymentForMismatch(order.OrderId, tx.TxHash().String(), fromAddr, decValue, pc.PayTokenId)
		return false, nil
	}
	// update payment info
	if err = pc.DoPayment(order, tx.TxHash().String(), fromAddr, pc.ParserType.ToAlgorithmId()); err != nil {
		return false, fmt.Errorf("pc.DoPayment err: %s", err.Error())
	}

	return true, nil
}

func (p *ParserBtc) dealWithHashAndAmount(pc *parser_common.ParserCore, tx *wire.MsgTx, decValue decimal.Decimal, fromAddr, toAddr string) error {
	var order tables.TableOrderInfo
	var err error

	order, err = pc.DbDao.GetOrderByAddrWithAmountAndAddr(fromAddr, toAddr, pc.PayTokenId, decValue)
	if err != nil {
		return fmt.Errorf("GetOrderByAddrWithAmountAndAddr err: %s", err.Error())
	}
	log.Info("dealWithHashAndAmount:", tx.TxHash().String(), order.OrderId)
	if order.Id > 0 {
		if err = pc.DoPayment(order, tx.TxHash().String(), fromAddr, pc.ParserType.ToAlgorithmId()); err != nil {
			return fmt.Errorf("pc.DoPayment err: %s", err.Error())
		}
	} else {
		//pc.CreatePaymentForMismatch("", data.Txid, addrPayload, decValue, pc.PayTokenId)
		msg := `hash: %s
addrPayload: %s`
		notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "dealWithHashAndAmount", fmt.Sprintf(msg, tx.TxHash().String(), fromAddr))
	}
	return nil
}
