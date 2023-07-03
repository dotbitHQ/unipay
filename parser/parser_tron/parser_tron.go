package parser_tron

import (
	"encoding/hex"
	"fmt"
	"github.com/dotbitHQ/das-lib/chain/chain_tron"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/golang/protobuf/proto"
	"github.com/scorpiotzh/mylog"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
	"math/big"
	"strings"
	"sync"
	"unipay/parser/parser_common"
	"unipay/tables"
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
	parserType, payTokenId := pc.ParserType, pc.PayTokenId
	if block == nil {
		return fmt.Errorf("block is nil")
	}
	contractUSDT, contractPayTokenId := pc.ContractAddress, pc.ContractPayTokenId
	for _, tx := range block.Transactions {
		if len(tx.Transaction.RawData.Contract) != 1 {
			continue
		}

		switch tx.Transaction.RawData.Contract[0].Type {
		case core.Transaction_Contract_TransferContract:
			instance := core.TransferContract{}
			if err := proto.Unmarshal(tx.Transaction.RawData.Contract[0].Parameter.Value, &instance); err != nil {
				log.Error(" proto.Unmarshal err:", err.Error())
				continue
			}
			orderId := chain_tron.GetMemo(tx.Transaction.RawData.Data)
			fromAddr, toAddr := hex.EncodeToString(instance.OwnerAddress), hex.EncodeToString(instance.ToAddress)
			if _, ok := pc.AddrMap[toAddr]; !ok {
				continue
			}
			log.Info("parsingBlockData:", parserType, orderId, hex.EncodeToString(tx.Txid))
			if orderId == "" {
				continue
			} else if len(orderId) > 64 {
				continue
			}
			amountValue := decimal.New(instance.Amount, 0)
			// check order id
			order, err := pc.DbDao.GetOrderInfoByOrderIdWithAddr(orderId, toAddr)
			if err != nil {
				return fmt.Errorf("GetOrderInfoByOrderIdWithAddr err: %s", err.Error())
			} else if order.Id == 0 {
				pc.CreatePaymentForMismatch(common.DasAlgorithmIdTron, "", hex.EncodeToString(tx.Txid), fromAddr, amountValue, payTokenId)
				log.Warn("GetOrderInfoByOrderId is not exist:", parserType, orderId)
				continue
			}
			if order.PayTokenId != payTokenId {
				log.Warn("order pay token id not match", order.OrderId)
				pc.CreatePaymentForMismatch(common.DasAlgorithmIdTron, order.OrderId, hex.EncodeToString(tx.Txid), fromAddr, amountValue, payTokenId)
				continue
			}
			if amountValue.Cmp(order.Amount) == -1 {
				log.Warn("tx value less than order amount:", amountValue.String(), order.Amount.String())
				pc.CreatePaymentForAmountMismatch(order, hex.EncodeToString(tx.Txid), fromAddr, amountValue)
				continue
			}
			// change the status to confirm
			if err = pc.DoPayment(order, hex.EncodeToString(tx.Txid), fromAddr); err != nil {
				return fmt.Errorf("pc.DoPayment err: %s", err.Error())
			}
		//case core.Transaction_Contract_TransferAssetContract:
		case core.Transaction_Contract_TriggerSmartContract:
			smart := core.TriggerSmartContract{}
			if err := proto.Unmarshal(tx.Transaction.RawData.Contract[0].Parameter.Value, &smart); err != nil {
				log.Error(" proto.Unmarshal err:", err.Error())
				continue
			}
			fromHex, contractHex := hex.EncodeToString(smart.OwnerAddress), hex.EncodeToString(smart.ContractAddress)
			if contractHex != contractUSDT {
				continue
			}
			data := hex.EncodeToString(smart.Data)
			toHex := common.TronPreFix + hex.EncodeToString(smart.Data[16:36])
			amount := decimal.NewFromBigInt(new(big.Int).SetBytes(smart.Data[36:]), 0)

			log.Info("parsingBlockData:", fromHex, contractHex, toHex, amount.String())
			if len(smart.Data) != 68 || !strings.Contains(data, "a9059cbb0000") {
				continue
			}
			if _, ok := pc.AddrMap[toHex]; !ok {
				continue
			}
			order, err := pc.DbDao.GetOrderByAddrWithAmountAndAddr(fromHex, toHex, contractPayTokenId, amount)
			if err != nil {
				return fmt.Errorf("GetOrderByAddrWithAmountAndAddr err: %s", err.Error())
			} else if order.Id == 0 {
				log.Warn("order not exist:", contractPayTokenId, fromHex, amount)
				pc.CreatePaymentForMismatch(common.DasAlgorithmIdTron, "", hex.EncodeToString(tx.Txid), fromHex, amount, contractPayTokenId)
				continue
			}
			if order.PayTokenId != contractPayTokenId {
				log.Warn("order pay token id not match", order.OrderId, order.PayTokenId, contractPayTokenId)
				pc.CreatePaymentForMismatch(common.DasAlgorithmIdTron, order.OrderId, hex.EncodeToString(tx.Txid), fromHex, amount, contractPayTokenId)
				continue
			}
			if err = pc.DoPayment(order, hex.EncodeToString(tx.Txid), fromHex); err != nil {
				return fmt.Errorf("pc.DoPayment err: %s", err.Error())
			}
		}
	}
	return nil
}
