package parser_bitcoin

import (
	"fmt"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/txscript"
	"github.com/dotbitHQ/das-lib/bitcoin"
	"github.com/dotbitHQ/unipay/config"
	"github.com/dotbitHQ/unipay/notify"
	"github.com/dotbitHQ/unipay/parser/parser_common"
	"github.com/dotbitHQ/unipay/tables"
	"github.com/scorpiotzh/mylog"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
	"sync"
	"sync/atomic"
	"time"
)

var log = mylog.NewLogger("parser_bitcoin", mylog.LevelDebug)

type ParserBitcoin struct {
	parser_common.ParserCommon
	NodeRpc *bitcoin.BaseRequest
}

func (p *ParserBitcoin) getLatestBlockNumber() (uint64, error) {
	data, err := p.NodeRpc.GetBlockChainInfo()
	if err != nil {
		return 0, fmt.Errorf("GetBlockChainInfo err: %s", err.Error())
	}
	return data.Blocks, nil
}

func (p *ParserBitcoin) Parser() {
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

func (p *ParserBitcoin) parserConcurrencyMode() error {
	log.Info("parserConcurrencyMode:", p.ParserType, p.CurrentBlockNumber, p.ConcurrencyNum)

	var blockList = make([]tables.TableBlockParserInfo, p.ConcurrencyNum)
	var blocks = make([]bitcoin.BlockInfo, p.ConcurrencyNum)
	var blockCh = make(chan bitcoin.BlockInfo, p.ConcurrencyNum)

	blockLock := &sync.Mutex{}
	blockGroup := &errgroup.Group{}

	for i := uint64(0); i < p.ConcurrencyNum; i++ {
		bn := p.CurrentBlockNumber + i
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
			if err := p.parsingBlockData(&v); err != nil {
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
func (p *ParserBitcoin) parserSubMode() error {
	log.Info("parserSubMode:", p.ParserType, p.CurrentBlockNumber)

	hash, err := p.NodeRpc.GetBlockHash(p.CurrentBlockNumber)
	if err != nil {
		return fmt.Errorf("req GetBlockHash err: %s", err.Error())
	}

	block, err := p.NodeRpc.GetBlock(hash)
	if err != nil {
		return fmt.Errorf("req GetBlock err: %s", err.Error())
	}
	blockHash := block.Hash
	parentHash := block.PreviousBlockHash
	log.Info("parserSubMode:", p.ParserType, blockHash, parentHash)
	if fork, err := p.CheckFork(parentHash); err != nil {
		return fmt.Errorf("CheckFork err: %s", err.Error())
	} else if fork {
		log.Warn("CheckFork is true:", p.ParserType, p.CurrentBlockNumber, blockHash, parentHash)
		if err := p.DbDao.DeleteBlockInfoByBlockNumber(p.ParserType, p.CurrentBlockNumber-1); err != nil {
			return fmt.Errorf("DeleteBlockInfoByBlockNumber err: %s", err.Error())
		}
		atomic.AddUint64(&p.CurrentBlockNumber, ^uint64(0))
	} else if err := p.parsingBlockData(&block); err != nil {
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

func (p *ParserBitcoin) parsingBlockData(block *bitcoin.BlockInfo) error {
	if block == nil {
		return fmt.Errorf("block is nil")
	}
	log.Info("parsingBlockData:", p.ParserType, block.Height, block.Hash, len(block.Tx))
	for _, v := range block.Tx {
		// get tx info
		data, err := p.NodeRpc.GetRawTransaction(v)
		if err != nil {
			return fmt.Errorf("req GetRawTransaction err: %s", err.Error())
		}
		// check address of outputs
		isMyTx, value := false, float64(0)
		for _, vOut := range data.Vout {
			for _, addr := range vOut.ScriptPubKey.Addresses {
				if addr == p.Address {
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
			log.Info("parsingBlockData:", p.ParserType, v)
			if len(data.Vin) == 0 {
				return fmt.Errorf("tx vin is nil")
			}
			_, addrPayload, err := bitcoin.VinScriptSigToAddress(data.Vin[0].ScriptSig, bitcoin.GetDogeMainNetParams())
			if err != nil {
				return fmt.Errorf("VinScriptSigToAddress err: %s", err.Error())
			}

			if ok, err := p.dealWithOpReturn(data, decValue, addrPayload); err != nil {
				return fmt.Errorf("dealWithOpReturn err: %s", err.Error())
			} else if ok {
				continue
			}
			if err = p.dealWithHashAndAmount(data, decValue, addrPayload); err != nil {
				return fmt.Errorf("dealWithHashAndAmount err: %s", err.Error())
			}
		}
	}
	return nil
}

func (p *ParserBitcoin) dealWithOpReturn(data btcjson.TxRawResult, decValue decimal.Decimal, addrPayload string) (bool, error) {
	var orderId string
	for _, vOut := range data.Vout {
		switch vOut.ScriptPubKey.Type {
		case txscript.NullDataTy.String():
			if lenHex := len(vOut.ScriptPubKey.Hex); lenHex > 32 {
				orderId = vOut.ScriptPubKey.Hex[lenHex-32:]
				break
			}
		}
	}
	log.Info("checkOpReturn:", orderId, addrPayload)
	if orderId == "" {
		return false, nil
	}
	order, err := p.DbDao.GetOrderInfoByOrderId(orderId)
	if err != nil {
		return false, fmt.Errorf("GetOrderInfoByOrderId err: %s", err.Error())
	} else if order.Id == 0 {
		log.Warn("order not exist:", p.ParserType, orderId)
		return false, nil
	}
	if order.PayTokenId != p.PayTokenId {
		log.Warn("order pay token id not match", order.OrderId)
		return false, nil
	}
	payAmount := order.Amount.DivRound(decimal.NewFromInt(1e8), 8)
	if payAmount.Cmp(decValue) != 0 {
		log.Warn("tx value not match order amount:", decValue.String(), payAmount.String())
		return false, nil
	}
	// update payment info
	paymentInfo := tables.TablePaymentInfo{
		Id:            0,
		PayHash:       data.Txid,
		OrderId:       order.OrderId,
		PayAddress:    addrPayload,
		AlgorithmId:   order.AlgorithmId,
		Timestamp:     data.Blocktime,
		Amount:        order.Amount,
		PayHashStatus: tables.PayHashStatusConfirm,
		RefundStatus:  tables.RefundStatusDefault,
		RefundHash:    "",
		RefundNonce:   0,
	}
	if err := p.DbDao.UpdatePaymentStatus(paymentInfo); err != nil {
		return false, fmt.Errorf("UpdatePaymentStatus err: %s", err.Error())
	}

	return true, nil
}

func (p *ParserBitcoin) dealWithHashAndAmount(data btcjson.TxRawResult, decValue decimal.Decimal, addrPayload string) error {
	var order tables.TableOrderInfo
	var err error

	decValue = decValue.Mul(decimal.NewFromInt(1e8))
	order, err = p.DbDao.GetOrderByAddrWithAmount(addrPayload, p.PayTokenId, decValue)
	if err != nil {
		return fmt.Errorf("GetOrderByAddrWithAmount err: %s", err.Error())
	}
	log.Info("dealWithHashAndAmount:", data.Txid, order.OrderId)
	if order.Id > 0 {
		paymentInfo := tables.TablePaymentInfo{
			Id:            0,
			PayHash:       data.Txid,
			OrderId:       order.OrderId,
			PayAddress:    addrPayload,
			AlgorithmId:   order.AlgorithmId,
			Timestamp:     data.Blocktime,
			Amount:        order.Amount,
			PayHashStatus: tables.PayHashStatusConfirm,
			RefundStatus:  tables.RefundStatusDefault,
			RefundHash:    "",
			RefundNonce:   0,
		}
		if err := p.DbDao.UpdatePaymentStatus(paymentInfo); err != nil {
			return fmt.Errorf("UpdatePaymentStatus err: %s", err.Error())
		}
	} else {
		msg := `hash: %s
addrPayload: %s`
		notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "dealWithHashAndAmount", fmt.Sprintf(msg, data.Txid, addrPayload))
	}
	return nil
}
