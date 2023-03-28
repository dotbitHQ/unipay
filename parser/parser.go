package parser

import (
	"context"
	"fmt"
	"github.com/dotbitHQ/das-lib/bitcoin"
	"github.com/dotbitHQ/das-lib/chain/chain_evm"
	"github.com/dotbitHQ/das-lib/chain/chain_tron"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/unipay/config"
	"github.com/dotbitHQ/unipay/dao"
	"github.com/dotbitHQ/unipay/parser/parser_bitcoin"
	"github.com/dotbitHQ/unipay/parser/parser_ckb"
	"github.com/dotbitHQ/unipay/parser/parser_common"
	"github.com/dotbitHQ/unipay/parser/parser_evm"
	"github.com/dotbitHQ/unipay/parser/parser_tron"
	"github.com/dotbitHQ/unipay/tables"
	"github.com/nervosnetwork/ckb-sdk-go/rpc"
	"strings"
	"sync"
)

type ToolParser struct {
	ParserEth     *parser_evm.ParserEvm
	ParserBsc     *parser_evm.ParserEvm
	ParserPolygon *parser_evm.ParserEvm
	ParserCkb     *parser_ckb.ParserCkb
	ParserTron    *parser_tron.ParserTron
	ParserDoge    *parser_bitcoin.ParserBitcoin

	ctx   context.Context
	wg    *sync.WaitGroup
	dbDao *dao.DbDao
}

func NewToolParser(ctx context.Context, wg *sync.WaitGroup, dbDao *dao.DbDao) (*ToolParser, error) {
	kp := ToolParser{
		ParserEth:     nil,
		ParserBsc:     nil,
		ParserPolygon: nil,
		ParserCkb:     nil,
		ParserTron:    nil,
		ParserDoge:    nil,
		ctx:           ctx,
		wg:            wg,
		dbDao:         dbDao,
	}

	if err := kp.initParserEth(); err != nil {
		return nil, fmt.Errorf("initParserEth err: %s", err.Error())
	}
	if err := kp.initParserBsc(); err != nil {
		return nil, fmt.Errorf("initParserBsc err: %s", err.Error())
	}
	if err := kp.initParserPolygon(); err != nil {
		return nil, fmt.Errorf("initParserPolygon err: %s", err.Error())
	}
	if err := kp.initParserTron(); err != nil {
		return nil, fmt.Errorf("initParserTron err: %s", err.Error())
	}
	if err := kp.initParserCkb(); err != nil {
		return nil, fmt.Errorf("initParserCkb err: %s", err.Error())
	}
	if err := kp.initParserDoge(); err != nil {
		return nil, fmt.Errorf("initParserDoge err: %s", err.Error())
	}
	return &kp, nil
}

func (t *ToolParser) initParserEth() error {
	if !config.Cfg.Chain.Eth.Switch {
		return nil
	}
	chainEvm, err := chain_evm.Initialize(t.ctx, config.Cfg.Chain.Eth.Node, config.Cfg.Chain.Eth.RefundAddFee)
	if err != nil {
		return fmt.Errorf("chain_evm.Initialize eth err: %s", err.Error())
	}
	t.ParserEth = &parser_evm.ParserEvm{
		ParserCommon: parser_common.ParserCommon{
			Ctx:                t.ctx,
			Wg:                 t.wg,
			DbDao:              t.dbDao,
			ParserType:         tables.ParserTypeETH,
			PayTokenId:         tables.PayTokenIdETH,
			Address:            config.Cfg.Chain.Eth.Address,
			CurrentBlockNumber: 0,
			ConcurrencyNum:     5,
			ConfirmNum:         2,
		},
		ChainEvm: chainEvm,
	}
	return nil
}

func (t *ToolParser) initParserBsc() error {
	if !config.Cfg.Chain.Bsc.Switch {
		return nil
	}
	chainEvm, err := chain_evm.Initialize(t.ctx, config.Cfg.Chain.Bsc.Node, config.Cfg.Chain.Bsc.RefundAddFee)
	if err != nil {
		return fmt.Errorf("chain_evm.Initialize bsc err: %s", err.Error())
	}
	t.ParserBsc = &parser_evm.ParserEvm{
		ParserCommon: parser_common.ParserCommon{
			Ctx:                t.ctx,
			Wg:                 t.wg,
			DbDao:              t.dbDao,
			ParserType:         tables.ParserTypeBSC,
			PayTokenId:         tables.PayTokenIdBNB,
			Address:            config.Cfg.Chain.Bsc.Address,
			CurrentBlockNumber: 0,
			ConcurrencyNum:     10,
			ConfirmNum:         10,
		},
		ChainEvm: chainEvm,
	}
	return nil
}

func (t *ToolParser) initParserPolygon() error {
	if !config.Cfg.Chain.Polygon.Switch {
		return nil
	}
	chainEvm, err := chain_evm.Initialize(t.ctx, config.Cfg.Chain.Polygon.Node, config.Cfg.Chain.Polygon.RefundAddFee)
	if err != nil {
		return fmt.Errorf("chain_evm.Initialize bsc err: %s", err.Error())
	}
	t.ParserBsc = &parser_evm.ParserEvm{
		ParserCommon: parser_common.ParserCommon{
			Ctx:                t.ctx,
			Wg:                 t.wg,
			DbDao:              t.dbDao,
			ParserType:         tables.ParserTypePOLYGON,
			PayTokenId:         tables.PayTokenIdMATIC,
			Address:            config.Cfg.Chain.Polygon.Address,
			CurrentBlockNumber: 0,
			ConcurrencyNum:     10,
			ConfirmNum:         10,
		},
		ChainEvm: chainEvm,
	}
	return nil
}

func (t *ToolParser) initParserTron() error {
	if !config.Cfg.Chain.Tron.Switch {
		return nil
	}
	chainTron, err := chain_tron.Initialize(t.ctx, config.Cfg.Chain.Tron.Node)
	if err != nil {
		return fmt.Errorf("chain_ckb.Initialize tron err: %s", err.Error())
	}
	address := config.Cfg.Chain.Tron.Address
	if strings.HasPrefix(address, common.TronBase58PreFix) {
		if address, err = common.TronBase58ToHex(address); err != nil {
			return fmt.Errorf("TronBase58ToHex err: %s", err.Error())
		}
	}
	t.ParserTron = &parser_tron.ParserTron{
		ParserCommon: parser_common.ParserCommon{
			Ctx:                t.ctx,
			Wg:                 t.wg,
			DbDao:              t.dbDao,
			ParserType:         tables.ParserTypeTRON,
			PayTokenId:         tables.PayTokenIdTRX,
			Address:            address,
			CurrentBlockNumber: 0,
			ConcurrencyNum:     10,
			ConfirmNum:         10,
		},
		ChainTron: chainTron,
	}
	return nil
}

func (t *ToolParser) initParserCkb() error {
	if !config.Cfg.Chain.Ckb.Switch {
		return nil
	}
	rpcClient, err := rpc.DialWithIndexer(config.Cfg.Chain.Ckb.Node, config.Cfg.Chain.Ckb.Node)
	if err != nil {
		return fmt.Errorf("rpc.DialWithIndexer err:%s", err.Error())
	}
	t.ParserCkb = &parser_ckb.ParserCkb{
		ParserCommon: parser_common.ParserCommon{
			Ctx:                t.ctx,
			Wg:                 t.wg,
			DbDao:              t.dbDao,
			ParserType:         tables.ParserTypeCKB,
			PayTokenId:         tables.PayTokenIdDAS,
			Address:            config.Cfg.Chain.Ckb.Address,
			CurrentBlockNumber: 0,
			ConcurrencyNum:     10,
			ConfirmNum:         2,
		},
		Client: rpcClient,
	}
	return nil
}

func (t *ToolParser) initParserDoge() error {
	if !config.Cfg.Chain.Doge.Switch {
		return nil
	}
	nodeRpc := bitcoin.BaseRequest{
		RpcUrl:   config.Cfg.Chain.Doge.Node,
		User:     config.Cfg.Chain.Doge.User,
		Password: config.Cfg.Chain.Doge.Password,
		Proxy:    "",
	}
	t.ParserDoge = &parser_bitcoin.ParserBitcoin{
		ParserCommon: parser_common.ParserCommon{
			Ctx:                t.ctx,
			Wg:                 t.wg,
			DbDao:              t.dbDao,
			ParserType:         tables.ParserTypeDoge,
			PayTokenId:         tables.PayTokenIdDOGE,
			Address:            config.Cfg.Chain.Doge.Address,
			CurrentBlockNumber: 0,
			ConcurrencyNum:     10,
			ConfirmNum:         3,
		},
		NodeRpc: &nodeRpc,
	}
	return nil
}

func (t *ToolParser) Run() {
	if config.Cfg.Chain.Ckb.Switch {
		go t.ParserCkb.Parser()
	}
	if config.Cfg.Chain.Eth.Switch {
		go t.ParserEth.Parser()
	}
	if config.Cfg.Chain.Bsc.Switch {
		go t.ParserBsc.Parser()
	}
	if config.Cfg.Chain.Polygon.Switch {
		go t.ParserPolygon.Parser()
	}
	if config.Cfg.Chain.Tron.Switch {
		go t.ParserTron.Parser()
	}
	if config.Cfg.Chain.Doge.Switch {
		go t.ParserDoge.Parser()
	}
}
