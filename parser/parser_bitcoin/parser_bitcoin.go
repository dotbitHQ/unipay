package parser_bitcoin

import (
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/dotbitHQ/das-lib/bitcoin"
	"github.com/scorpiotzh/mylog"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
	"strings"
	"sync"
	"unipay/config"
	"unipay/notify"
	"unipay/parser/parser_common"
	"unipay/tables"
)

var log = mylog.NewLogger("parser_bitcoin", mylog.LevelDebug)

type ParserBitcoin struct {
	NodeRpc *bitcoin.BaseRequest
}

func (p *ParserBitcoin) GetLatestBlockNumber() (uint64, error) {
	data, err := p.NodeRpc.GetBlockChainInfo()
	if err != nil {
		return 0, fmt.Errorf("GetBlockChainInfo err: %s", err.Error())
	}
	return data.Blocks, nil
}
func (p *ParserBitcoin) Init(pc *parser_common.ParserCore) error {
	return nil
}
func (p *ParserBitcoin) SingleParsing(pc *parser_common.ParserCore) error {
	parserType, currentBlockNumber := pc.ParserType, pc.CurrentBlockNumber
	log.Info("SingleParsing:", parserType, currentBlockNumber)

	hash, err := p.NodeRpc.GetBlockHash(currentBlockNumber)
	if err != nil {
		return fmt.Errorf("req GetBlockHash err: %s", err.Error())
	}

	block, err := p.NodeRpc.GetBlock(hash)
	if err != nil {
		return fmt.Errorf("req GetBlock err: %s", err.Error())
	}

	blockHash := block.Hash
	parentHash := block.PreviousBlockHash
	log.Info("SingleParsing:", parserType, blockHash, parentHash)

	if isFork, err := pc.HandleFork(blockHash, parentHash); err != nil {
		return fmt.Errorf("HandleFork err: %s", err.Error())
	} else if isFork {
		return nil
	}
	if err := p.parsingBlockData(&block, pc); err != nil {
		return fmt.Errorf("parsingBlockData err: %s", err.Error())
	} else {
		if err := pc.HandleSingleParsingOK(blockHash, parentHash); err != nil {
			return fmt.Errorf("HandleSingleParsingOK err: %s", err.Error())
		}
	}
	return nil
}
func (p *ParserBitcoin) ConcurrentParsing(pc *parser_common.ParserCore) error {
	parserType, concurrencyNum, currentBlockNumber := pc.ParserType, pc.ConcurrencyNum, pc.CurrentBlockNumber
	log.Info("ConcurrentParsing:", parserType, concurrencyNum, currentBlockNumber)

	var blockList = make([]tables.TableBlockParserInfo, concurrencyNum)
	var blocks = make([]bitcoin.BlockInfo, concurrencyNum)
	var blockCh = make(chan bitcoin.BlockInfo, concurrencyNum)

	blockLock := &sync.Mutex{}
	blockGroup := &errgroup.Group{}

	for i := uint64(0); i < concurrencyNum; i++ {
		bn := currentBlockNumber + i
		index := i
		blockGroup.Go(func() error {
			blockHash, err := p.NodeRpc.GetBlockHash(bn)
			if err != nil {
				return fmt.Errorf("req GetBlockHash err: %s", err.Error())
			}

			block, err := p.NodeRpc.GetBlock(blockHash)
			if err != nil {
				return fmt.Errorf("req GetBlock err: %s", err.Error())
			}

			hash := block.Hash
			parentHash := block.PreviousBlockHash

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
			if err := p.parsingBlockData(&v, pc); err != nil {
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

func (p *ParserBitcoin) getMainNetParams(pc *parser_common.ParserCore) (chaincfg.Params, error) {
	switch pc.ParserType {
	case tables.ParserTypeDoge:
		return bitcoin.GetDogeMainNetParams(), nil
	}
	return chaincfg.MainNetParams, fmt.Errorf("unknow MainNetParams ParserType[%d]", pc.ParserType)
}

func (p *ParserBitcoin) parsingBlockData(block *bitcoin.BlockInfo, pc *parser_common.ParserCore) error {
	parserType := pc.ParserType
	if block == nil {
		return fmt.Errorf("block is nil")
	}
	log.Info("parsingBlockData:", parserType, block.Height, block.Hash, len(block.Tx))
	for _, v := range block.Tx {
		// get tx info
		data, err := p.NodeRpc.GetRawTransaction(v)
		if err != nil {
			return fmt.Errorf("req GetRawTransaction err: %s", err.Error())
		}
		// check address of outputs
		isMyTx, value, receiptAddr := false, float64(0), ""
		for _, vOut := range data.Vout {
			for _, receiptAddr = range vOut.ScriptPubKey.Addresses {
				if _, ok := pc.AddrMap[receiptAddr]; ok {
					isMyTx = true
					value = vOut.Value
					break
				}
			}
			if isMyTx {
				break
			}
		}
		decValue := decimal.NewFromFloat(value)
		// check inputs & pay info & order id
		if isMyTx {
			log.Info("parsingBlockData:", parserType, v)
			if len(data.Vin) == 0 {
				return fmt.Errorf("tx vin is nil")
			}
			mainNetParams, err := p.getMainNetParams(pc)
			if err != nil {
				return fmt.Errorf("getMainNetParams err: %s", err.Error())
			}
			_, addrPayload, err := bitcoin.VinScriptSigToAddress(data.Vin[0].ScriptSig, mainNetParams)
			if err != nil {
				return fmt.Errorf("VinScriptSigToAddress err: %s", err.Error())
			}

			if ok, err := p.dealWithOpReturn(pc, data, decValue, addrPayload, receiptAddr); err != nil {
				return fmt.Errorf("dealWithOpReturn err: %s", err.Error())
			} else if ok {
				continue
			}
			if err = p.dealWithHashAndAmount(pc, data, decValue, addrPayload, receiptAddr); err != nil {
				return fmt.Errorf("dealWithHashAndAmount err: %s", err.Error())
			}
		}
	}
	return nil
}

func (p *ParserBitcoin) dealWithOpReturn(pc *parser_common.ParserCore, data btcjson.TxRawResult, decValue decimal.Decimal, addrPayload, receiptAddr string) (bool, error) {
	var orderId string
	for _, vOut := range data.Vout {
		switch vOut.ScriptPubKey.Type {
		case txscript.NullDataTy.String():
			asm := vOut.ScriptPubKey.Asm
			orderId = strings.TrimPrefix(asm, "OP_RETURN ")
			if len(orderId) == 64 {
				bys, _ := hex.DecodeString(orderId)
				orderId = string(bys)
			}
			break
		}
	}
	log.Info("dealWithOpReturn:", orderId, addrPayload)
	if orderId == "" {
		return false, nil
	}
	order, err := pc.DbDao.GetOrderInfoByOrderIdWithAddr(orderId, receiptAddr)
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
	decValue = decValue.Mul(decimal.NewFromInt(1e8))
	if decValue.Cmp(order.Amount) == -1 {
		log.Warn("tx value less than order amount:", decValue.String(), order.Amount.String())
		pc.CreatePaymentForMismatch(order.OrderId, data.Txid, addrPayload, decValue, pc.PayTokenId)
		return false, nil
	}
	// update payment info
	if err = pc.DoPayment(order, data.Txid, addrPayload, pc.ParserType.ToAlgorithmId()); err != nil {
		return false, fmt.Errorf("pc.DoPayment err: %s", err.Error())
	}

	return true, nil
}

func (p *ParserBitcoin) dealWithHashAndAmount(pc *parser_common.ParserCore, data btcjson.TxRawResult, decValue decimal.Decimal, addrPayload, receiptAddr string) error {
	var order tables.TableOrderInfo
	var err error

	decValue = decValue.Mul(decimal.NewFromInt(1e8))
	order, err = pc.DbDao.GetOrderByAddrWithAmountAndAddr(addrPayload, receiptAddr, pc.PayTokenId, decValue)
	if err != nil {
		return fmt.Errorf("GetOrderByAddrWithAmountAndAddr err: %s", err.Error())
	}
	log.Info("dealWithHashAndAmount:", data.Txid, order.OrderId)
	if order.Id > 0 {
		if err = pc.DoPayment(order, data.Txid, addrPayload, pc.ParserType.ToAlgorithmId()); err != nil {
			return fmt.Errorf("pc.DoPayment err: %s", err.Error())
		}
	} else {
		//pc.CreatePaymentForMismatch("", data.Txid, addrPayload, decValue, pc.PayTokenId)
		msg := `hash: %s
addrPayload: %s`
		notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "dealWithHashAndAmount", fmt.Sprintf(msg, data.Txid, addrPayload))
	}
	return nil
}
