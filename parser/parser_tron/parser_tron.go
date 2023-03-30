package parser_tron

import (
	"encoding/hex"
	"fmt"
	"github.com/dotbitHQ/das-lib/chain/chain_tron"
	"github.com/dotbitHQ/unipay/parser/parser_common"
	"github.com/dotbitHQ/unipay/tables"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/golang/protobuf/proto"
	"github.com/scorpiotzh/mylog"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
	"sync"
)

var log = mylog.NewLogger("parser_tron", mylog.LevelDebug)

type ParserTron struct {
	ChainTron *chain_tron.ChainTron
}

func (p *ParserTron) Init(pc *parser_common.ParserCore) error {
	return nil
}
func (p *ParserTron) GetLatestBlockNumber() (uint64, error) {
	currentBlockNumber, err := p.ChainTron.GetBlockNumber()
	if err != nil {
		return 0, fmt.Errorf("GetBlockNumber err: %s", err.Error())
	}
	return uint64(currentBlockNumber), nil
}
func (p *ParserTron) SingleParsing(pc *parser_common.ParserCore) error {
	parserType, currentBlockNumber := pc.ParserType, pc.CurrentBlockNumber
	log.Info("SingleParsing:", parserType, currentBlockNumber)

	block, err := p.ChainTron.GetBlockByNumber(currentBlockNumber)
	if err != nil {
		return fmt.Errorf("GetBlockByNumber err: %s", err.Error())
	}
	if block.BlockHeader == nil {
		return fmt.Errorf("block.BlockHeader is nil[%d]", currentBlockNumber)
	} else if block.BlockHeader.RawData == nil {
		return fmt.Errorf("block.BlockHeader.RawData is nil[%d]", currentBlockNumber)
	}

	blockHash := hex.EncodeToString(block.Blockid)
	parentHash := hex.EncodeToString(block.BlockHeader.RawData.ParentHash)
	log.Info("SingleParsing:", parserType, blockHash, parentHash)

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
func (p *ParserTron) ConcurrentParsing(pc *parser_common.ParserCore) error {
	parserType, concurrencyNum, currentBlockNumber := pc.ParserType, pc.ConcurrencyNum, pc.CurrentBlockNumber
	log.Info("ConcurrentParsing:", parserType, concurrencyNum, currentBlockNumber)

	var blockList = make([]tables.TableBlockParserInfo, concurrencyNum)
	var blocks = make([]*api.BlockExtention, concurrencyNum)
	var blockCh = make(chan *api.BlockExtention, concurrencyNum)

	blockLock := &sync.Mutex{}
	blockGroup := &errgroup.Group{}

	for i := uint64(0); i < concurrencyNum; i++ {
		bn := currentBlockNumber + i
		index := i
		blockGroup.Go(func() error {
			block, err := p.ChainTron.GetBlockByNumber(bn)
			if err != nil {
				return fmt.Errorf("GetBlockByNumber err:%s [%d]", err.Error(), bn)
			}
			if block.BlockHeader == nil {
				return fmt.Errorf("block.BlockHeader is nil[%d]", bn)
			} else if block.BlockHeader.RawData == nil {
				return fmt.Errorf("block.BlockHeader.RawData is nil[%d]", bn)
			}

			hash := hex.EncodeToString(block.Blockid)
			parentHash := hex.EncodeToString(block.BlockHeader.RawData.ParentHash)

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

func (p *ParserTron) parsingBlockData(block *api.BlockExtention, pc *parser_common.ParserCore) error {
	parserType, payTokenId, addr := pc.ParserType, pc.PayTokenId, pc.Address
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
			if toAddr != addr {
				continue
			}
			log.Info("parsingBlockData orderId:", orderId, hex.EncodeToString(tx.Txid))

			// check order id
			order, err := pc.DbDao.GetOrderInfoByOrderId(orderId)
			if err != nil {
				return fmt.Errorf("GetOrderInfoByOrderId err: %s", err.Error())
			} else if order.Id == 0 {
				log.Warn("GetOrderInfoByOrderId is not exist:", parserType, orderId)
				continue
			}
			if order.PayTokenId != payTokenId {
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
			if err := pc.DbDao.UpdatePaymentStatus(paymentInfo); err != nil {
				return fmt.Errorf("UpdatePaymentStatus err: %s", err.Error())
			}
		case core.Transaction_Contract_TransferAssetContract:
		case core.Transaction_Contract_TriggerSmartContract:
		}
	}
	return nil
}