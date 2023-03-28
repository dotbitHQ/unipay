package parser_ckb

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/unipay/config"
	"github.com/dotbitHQ/unipay/notify"
	"github.com/dotbitHQ/unipay/parser/parser_common"
	"github.com/dotbitHQ/unipay/tables"
	"github.com/nervosnetwork/ckb-sdk-go/address"
	"github.com/nervosnetwork/ckb-sdk-go/rpc"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/mylog"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var log = mylog.NewLogger("parser_ckb", mylog.LevelDebug)

type ParserCkb struct {
	parser_common.ParserCommon
	Client rpc.Client

	addressArgs string
}

func (p *ParserCkb) getLatestBlockNumber() (uint64, error) {
	if blockNumber, err := p.Client.GetTipBlockNumber(p.Ctx); err != nil {
		return 0, fmt.Errorf("GetTipBlockNumber err: %s", err.Error())
	} else {
		return blockNumber, nil
	}
}

func (p *ParserCkb) Parser() {
	parseAdd, err := address.Parse(p.Address)
	if err != nil {
		log.Error("address.Parse err:", p.ParserType, err.Error())
		return
	}
	p.addressArgs = common.Bytes2Hex(parseAdd.Script.Args)

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

func (p *ParserCkb) parsingBlockData(block *types.Block) error {
	if block == nil {
		return fmt.Errorf("block is nil")
	}
	for _, tx := range block.Transactions {
		for i, v := range tx.Outputs {
			if p.addressArgs != common.Bytes2Hex(v.Lock.Args) {
				continue
			}
			orderId := string(tx.OutputsData[i])
			if orderId == "" {
				continue
			}
			log.Info("parsingBlockData:", orderId, tx.Hash.Hex())
			capacity, _ := decimal.NewFromString(strconv.FormatUint(v.Capacity, 10))
			order, err := p.DbDao.GetOrderInfoByOrderId(orderId)
			if err != nil {
				return fmt.Errorf("GetOrderInfoByOrderId err: %s", err.Error())
			} else if order.Id == 0 {
				log.Warn("order not exist:", p.ParserType, orderId)
				continue
			}
			if order.PayTokenId != tables.PayTokenIdCKB && order.PayTokenId != tables.PayTokenIdDAS {
				log.Warn("order pay token id not match", order.OrderId)
				continue
			}
			if capacity.Cmp(order.Amount) == -1 {
				log.Warn("tx value less than order amount:", capacity.String(), order.Amount.String())
				continue
			}
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
			// change the status to confirm
			paymentInfo := tables.TablePaymentInfo{
				Id:            0,
				PayHash:       tx.Hash.Hex(),
				OrderId:       order.OrderId,
				PayAddress:    fromAddr,
				AlgorithmId:   order.AlgorithmId,
				Timestamp:     int64(block.Header.Timestamp),
				Amount:        order.Amount,
				PayHashStatus: tables.PayHashStatusConfirm,
				RefundStatus:  tables.RefundStatusDefault,
				RefundHash:    "",
				RefundNonce:   0,
			}
			if err := p.DbDao.UpdatePaymentStatus(paymentInfo); err != nil {
				return fmt.Errorf("UpdatePaymentStatus err: %s", err.Error())
			}
			break
		}
	}
	return nil
}

func (p *ParserCkb) parserSubMode() error {
	log.Info("parserSubMode:", p.ParserType, p.CurrentBlockNumber)

	block, err := p.Client.GetBlockByNumber(p.Ctx, p.CurrentBlockNumber)
	if err != nil {
		return fmt.Errorf("GetBlockByNumber err: %s", err.Error())
	}
	blockHash := block.Header.Hash.Hex()
	parentHash := block.Header.ParentHash.Hex()
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

func (p *ParserCkb) parserConcurrencyMode() error {
	log.Info("parserConcurrencyMode:", p.ParserType, p.CurrentBlockNumber, p.ConcurrencyNum)

	var blockList = make([]tables.TableBlockParserInfo, p.ConcurrencyNum)
	var blocks = make([]*types.Block, p.ConcurrencyNum)
	var blockCh = make(chan *types.Block, p.ConcurrencyNum)

	blockLock := &sync.Mutex{}
	blockGroup := &errgroup.Group{}

	for i := uint64(0); i < p.ConcurrencyNum; i++ {
		bn := p.CurrentBlockNumber + i
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
