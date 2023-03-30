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
	"github.com/dotbitHQ/unipay/notify"
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
	ParserCommonMap map[tables.ParserType]*parser_common.ParserCommon

	ctx   context.Context
	wg    *sync.WaitGroup
	dbDao *dao.DbDao
	cn    *notify.CallbackNotice
}

func NewToolParser(ctx context.Context, wg *sync.WaitGroup, dbDao *dao.DbDao, cn *notify.CallbackNotice) (*ToolParser, error) {
	tp := ToolParser{
		ParserCommonMap: make(map[tables.ParserType]*parser_common.ParserCommon),
		ctx:             ctx,
		wg:              wg,
		dbDao:           dbDao,
		cn:              cn,
	}

	if err := tp.initParserEth(); err != nil {
		return nil, fmt.Errorf("initParserEth err: %s", err.Error())
	}
	if err := tp.initParserBsc(); err != nil {
		return nil, fmt.Errorf("initParserBsc err: %s", err.Error())
	}
	if err := tp.initParserPolygon(); err != nil {
		return nil, fmt.Errorf("initParserPolygon err: %s", err.Error())
	}
	if err := tp.initParserTron(); err != nil {
		return nil, fmt.Errorf("initParserTron err: %s", err.Error())
	}
	if err := tp.initParserCkb(); err != nil {
		return nil, fmt.Errorf("initParserCkb err: %s", err.Error())
	}
	if err := tp.initParserDoge(); err != nil {
		return nil, fmt.Errorf("initParserDoge err: %s", err.Error())
	}
	return &tp, nil
}

func (t *ToolParser) initParserEth() error {
	if !config.Cfg.Chain.Eth.Switch {
		return nil
	}
	chainEvm, err := chain_evm.NewChainEvm(t.ctx, config.Cfg.Chain.Eth.Node, config.Cfg.Chain.Eth.RefundAddFee)
	if err != nil {
		return fmt.Errorf("chain_evm.NewChainEvm eth err: %s", err.Error())
	}
	t.ParserCommonMap[tables.ParserTypeETH] = &parser_common.ParserCommon{
		PC: &parser_common.ParserCore{
			Ctx:                t.ctx,
			Wg:                 t.wg,
			DbDao:              t.dbDao,
			CN:                 t.cn,
			ParserType:         tables.ParserTypeETH,
			PayTokenId:         tables.PayTokenIdETH,
			Address:            config.Cfg.Chain.Eth.Address,
			CurrentBlockNumber: 0,
			ConcurrencyNum:     5,
			ConfirmNum:         2,
			Switch:             config.Cfg.Chain.Eth.Switch,
		},
		PA: &parser_evm.ParserEvm{
			ChainEvm: chainEvm,
		},
	}
	return nil
}

func (t *ToolParser) initParserBsc() error {
	if !config.Cfg.Chain.Bsc.Switch {
		return nil
	}
	chainEvm, err := chain_evm.NewChainEvm(t.ctx, config.Cfg.Chain.Bsc.Node, config.Cfg.Chain.Bsc.RefundAddFee)
	if err != nil {
		return fmt.Errorf("chain_evm.NewChainEvm bsc err: %s", err.Error())
	}
	t.ParserCommonMap[tables.ParserTypeBSC] = &parser_common.ParserCommon{
		PC: &parser_common.ParserCore{
			Ctx:                t.ctx,
			Wg:                 t.wg,
			DbDao:              t.dbDao,
			CN:                 t.cn,
			ParserType:         tables.ParserTypeBSC,
			PayTokenId:         tables.PayTokenIdBNB,
			Address:            config.Cfg.Chain.Bsc.Address,
			CurrentBlockNumber: 0,
			ConcurrencyNum:     10,
			ConfirmNum:         10,
			Switch:             config.Cfg.Chain.Bsc.Switch,
		},
		PA: &parser_evm.ParserEvm{
			ChainEvm: chainEvm,
		},
	}

	return nil
}

func (t *ToolParser) initParserPolygon() error {
	if !config.Cfg.Chain.Polygon.Switch {
		return nil
	}
	chainEvm, err := chain_evm.NewChainEvm(t.ctx, config.Cfg.Chain.Polygon.Node, config.Cfg.Chain.Polygon.RefundAddFee)
	if err != nil {
		return fmt.Errorf("chain_evm.NewChainEvm bsc err: %s", err.Error())
	}
	t.ParserCommonMap[tables.ParserTypePOLYGON] = &parser_common.ParserCommon{
		PC: &parser_common.ParserCore{
			Ctx:                t.ctx,
			Wg:                 t.wg,
			DbDao:              t.dbDao,
			CN:                 t.cn,
			ParserType:         tables.ParserTypePOLYGON,
			PayTokenId:         tables.PayTokenIdMATIC,
			Address:            config.Cfg.Chain.Polygon.Address,
			CurrentBlockNumber: 0,
			ConcurrencyNum:     10,
			ConfirmNum:         10,
			Switch:             config.Cfg.Chain.Polygon.Switch,
		},
		PA: &parser_evm.ParserEvm{
			ChainEvm: chainEvm,
		},
	}
	return nil
}

func (t *ToolParser) initParserTron() error {
	if !config.Cfg.Chain.Tron.Switch {
		return nil
	}
	chainTron, err := chain_tron.NewChainTron(t.ctx, config.Cfg.Chain.Tron.Node)
	if err != nil {
		return fmt.Errorf("chain_ckb.NewChainTron tron err: %s", err.Error())
	}
	address := config.Cfg.Chain.Tron.Address
	if strings.HasPrefix(address, common.TronBase58PreFix) {
		if address, err = common.TronBase58ToHex(address); err != nil {
			return fmt.Errorf("TronBase58ToHex err: %s", err.Error())
		}
	}
	t.ParserCommonMap[tables.ParserTypeTRON] = &parser_common.ParserCommon{
		PC: &parser_common.ParserCore{
			Ctx:                t.ctx,
			Wg:                 t.wg,
			DbDao:              t.dbDao,
			CN:                 t.cn,
			ParserType:         tables.ParserTypeTRON,
			PayTokenId:         tables.PayTokenIdTRX,
			Address:            address,
			CurrentBlockNumber: 0,
			ConcurrencyNum:     10,
			ConfirmNum:         10,
			Switch:             config.Cfg.Chain.Tron.Switch,
		},
		PA: &parser_tron.ParserTron{ChainTron: chainTron},
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
	t.ParserCommonMap[tables.ParserTypeCKB] = &parser_common.ParserCommon{
		PC: &parser_common.ParserCore{
			Ctx:                t.ctx,
			Wg:                 t.wg,
			DbDao:              t.dbDao,
			CN:                 t.cn,
			ParserType:         tables.ParserTypeCKB,
			PayTokenId:         tables.PayTokenIdDAS,
			Address:            config.Cfg.Chain.Ckb.Address,
			CurrentBlockNumber: 0,
			ConcurrencyNum:     10,
			ConfirmNum:         2,
			Switch:             config.Cfg.Chain.Ckb.Switch,
		},
		PA: &parser_ckb.ParserCkb{
			Ctx:    t.ctx,
			Client: rpcClient,
		},
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
	t.ParserCommonMap[tables.ParserTypeDoge] = &parser_common.ParserCommon{
		PC: &parser_common.ParserCore{
			Ctx:                t.ctx,
			Wg:                 t.wg,
			DbDao:              t.dbDao,
			CN:                 t.cn,
			ParserType:         tables.ParserTypeDoge,
			PayTokenId:         tables.PayTokenIdDOGE,
			Address:            config.Cfg.Chain.Doge.Address,
			CurrentBlockNumber: 0,
			ConcurrencyNum:     5,
			ConfirmNum:         3,
			Switch:             config.Cfg.Chain.Doge.Switch,
		},
		PA: &parser_bitcoin.ParserBitcoin{NodeRpc: &nodeRpc},
	}
	return nil
}

func (t *ToolParser) RunParser() {
	for _, v := range t.ParserCommonMap {
		if v.PC.Switch {
			go v.Parser()
		}
	}
}
