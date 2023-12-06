package parser_dp

import (
	"context"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/http_api/logger"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"golang.org/x/sync/errgroup"
	"sync"
	"unipay/parser/parser_common"
	"unipay/tables"
)

var log = logger.NewLogger("parser_dp", logger.LevelDebug)

type ParserDP struct {
	Ctx                  context.Context
	DasCore              *core.DasCore
	mapTransactionHandle map[common.DasAction]FuncTransactionHandle
}

func (p *ParserDP) Init(pc *parser_common.ParserCore) error {
	p.registerTransactionHandle()
	return nil
}
func (p *ParserDP) GetLatestBlockNumber() (uint64, error) {
	if blockNumber, err := p.DasCore.Client().GetTipBlockNumber(p.Ctx); err != nil {
		return 0, fmt.Errorf("GetTipBlockNumber err: %s", err.Error())
	} else {
		return blockNumber, nil
	}
}
func (p *ParserDP) SingleParsing(pc *parser_common.ParserCore) error {
	parserType, currentBlockNumber := pc.ParserType, pc.CurrentBlockNumber
	log.Info("SingleParsing:", parserType, currentBlockNumber)

	block, err := p.DasCore.Client().GetBlockByNumber(p.Ctx, currentBlockNumber)
	if err != nil {
		return fmt.Errorf("GetBlockByNumber err: %s", err.Error())
	}

	blockHash := block.Header.Hash.Hex()
	parentHash := block.Header.ParentHash.Hex()
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
func (p *ParserDP) ConcurrentParsing(pc *parser_common.ParserCore) error {
	parserType, concurrencyNum, currentBlockNumber := pc.ParserType, pc.ConcurrencyNum, pc.CurrentBlockNumber
	log.Info("ConcurrentParsing:", parserType, concurrencyNum, currentBlockNumber)

	var blockList = make([]tables.TableBlockParserInfo, concurrencyNum)
	var blocks = make([]*types.Block, concurrencyNum)
	var blockCh = make(chan *types.Block, concurrencyNum)

	blockLock := &sync.Mutex{}
	blockGroup := &errgroup.Group{}

	for i := uint64(0); i < concurrencyNum; i++ {
		bn := currentBlockNumber + i
		index := i
		blockGroup.Go(func() error {
			block, err := p.DasCore.Client().GetBlockByNumber(p.Ctx, bn)
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

func (p *ParserDP) parsingBlockData(block *types.Block, pc *parser_common.ParserCore) error {
	parserType := pc.ParserType
	if block == nil {
		return fmt.Errorf("block is nil")
	}
	log.Debug("parsingBlockData:", parserType, block.Header.Number)
	for _, tx := range block.Transactions {
		txHash := tx.Hash.Hex()
		blockNumber := block.Header.Number
		blockTimestamp := block.Header.Timestamp
		//log.Info("parsingBlockData:", tx.Hash.Hex())

		if builder, err := witness.ActionDataBuilderFromTx(tx); err != nil {
			//log.Warn("ActionDataBuilderFromTx err:", err.Error())
		} else {
			log.Info("ActionDataBuilderFromTx:", builder.Action, tx.Hash.Hex())
			if handle, ok := p.mapTransactionHandle[builder.Action]; ok {
				// transaction parse by action
				resp := handle(FuncTransactionHandleReq{
					DbDao:          pc.DbDao,
					Tx:             tx,
					TxHash:         txHash,
					BlockNumber:    blockNumber,
					BlockTimestamp: int64(blockTimestamp),
					Action:         builder.Action,
				}, pc)
				if resp.Err != nil {
					// todo
					log.Error("action handle resp:", builder.Action, blockNumber, txHash, resp.Err.Error())
					return resp.Err
				}
			}
		}
	}
	return nil
}
