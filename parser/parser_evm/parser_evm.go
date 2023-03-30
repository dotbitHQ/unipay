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
)

var log = mylog.NewLogger("parser_evm", mylog.LevelDebug)

type ParserEvm struct {
	ChainEvm *chain_evm.ChainEvm
}

func (p *ParserEvm) Init(pc *parser_common.ParserCore) error {
	return nil
}
func (p *ParserEvm) GetLatestBlockNumber() (uint64, error) {
	currentBlockNumber, err := p.ChainEvm.BestBlockNumber()
	if err != nil {
		return 0, fmt.Errorf("BestBlockNumber err: %s", err.Error())
	}
	return currentBlockNumber, nil
}
func (p *ParserEvm) SingleParsing(pc *parser_common.ParserCore) error {
	parserType, currentBlockNumber := pc.ParserType, pc.CurrentBlockNumber
	log.Info("SingleParsing:", parserType, currentBlockNumber)

	block, err := p.ChainEvm.GetBlockByNumber(currentBlockNumber)
	if err != nil {
		return fmt.Errorf("GetBlockByNumber err: %s", err.Error())
	}
	if block.Hash == "" || block.ParentHash == "" {
		log.Info("GetBlockByNumber:", currentBlockNumber, toolib.JsonString(&block))
		return fmt.Errorf("GetBlockByNumber data is nil: [%d]", currentBlockNumber)
	}

	blockHash := block.Hash
	parentHash := block.ParentHash
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
func (p *ParserEvm) ConcurrentParsing(pc *parser_common.ParserCore) error {
	parserType, concurrencyNum, currentBlockNumber := pc.ParserType, pc.ConcurrencyNum, pc.CurrentBlockNumber
	log.Info("ConcurrentParsing:", parserType, concurrencyNum, currentBlockNumber)

	var blockList = make([]tables.TableBlockParserInfo, concurrencyNum)
	var blocks = make([]*chain_evm.Block, concurrencyNum)
	var blockCh = make(chan *chain_evm.Block, concurrencyNum)

	blockLock := &sync.Mutex{}
	blockGroup := &errgroup.Group{}

	for i := uint64(0); i < concurrencyNum; i++ {
		bn := currentBlockNumber + i
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

func (p *ParserEvm) parsingBlockData(block *chain_evm.Block, pc *parser_common.ParserCore) error {
	parserType, payTokenId, addr := pc.ParserType, pc.PayTokenId, pc.Address
	if block == nil {
		return fmt.Errorf("block is nil")
	}
	for _, tx := range block.Transactions {
		switch strings.ToLower(ethcommon.HexToAddress(tx.To).Hex()) {
		case strings.ToLower(addr):
			orderId := string(ethcommon.FromHex(tx.Input))
			log.Info("parsingBlockData:", parserType, tx.Hash, tx.From, orderId, tx.Value)
			if orderId == "" {
				continue
			}
			// select order by order id which in tx memo
			order, err := pc.DbDao.GetOrderInfoByOrderId(orderId)
			if err != nil {
				return fmt.Errorf("GetOrderInfoByOrderId err: %s", err.Error())
			} else if order.Id == 0 {
				log.Warn("order not exist:", parserType, orderId)
				continue
			}
			if order.PayTokenId != payTokenId {
				log.Warn("order pay token id not match", order.OrderId, payTokenId)
				continue
			}
			// check value is equal amount or not
			decValue := decimal.NewFromBigInt(chain_evm.BigIntFromHex(tx.Value), 0)
			if decValue.Cmp(order.Amount) == -1 {
				log.Warn("tx value less than order amount:", parserType, decValue, order.Amount.String())
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
			if err := pc.DbDao.UpdatePaymentStatus(paymentInfo); err != nil {
				return fmt.Errorf("UpdatePaymentStatus err: %s", err.Error())
			}
		}
		continue
	}
	return nil
}
