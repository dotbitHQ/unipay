package parser

import (
	"context"
	"fmt"
	"github.com/dotbitHQ/das-lib/bitcoin"
	"github.com/dotbitHQ/das-lib/chain/chain_evm"
	"github.com/dotbitHQ/das-lib/chain/chain_tron"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/nervosnetwork/ckb-sdk-go/rpc"
	"sync"
	"unipay/config"
	"unipay/dao"
	"unipay/notify"
	"unipay/parser/parser_bitcoin"
	"unipay/parser/parser_ckb"
	"unipay/parser/parser_common"
	"unipay/parser/parser_dp"
	"unipay/parser/parser_evm"
	"unipay/parser/parser_tron"
	"unipay/tables"
)

type ToolParser struct {
	ctx     context.Context
	wg      *sync.WaitGroup
	dbDao   *dao.DbDao
	dasCore *core.DasCore

	parserCommonMap map[tables.ParserType]*parser_common.ParserCommon

	cn *notify.CallbackNotice
}

func NewToolParser(ctx context.Context, wg *sync.WaitGroup, dbDao *dao.DbDao, cn *notify.CallbackNotice, dasCore *core.DasCore) (*ToolParser, error) {
	tp := ToolParser{
		parserCommonMap: make(map[tables.ParserType]*parser_common.ParserCommon),
		ctx:             ctx,
		wg:              wg,
		dbDao:           dbDao,
		cn:              cn,
		dasCore:         dasCore,
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
	if err := tp.initParserDP(); err != nil {
		return nil, fmt.Errorf("initParserDP err: %s", err.Error())
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
	t.parserCommonMap[tables.ParserTypeETH] = &parser_common.ParserCommon{
		PC: &parser_common.ParserCore{
			Ctx:                t.ctx,
			Wg:                 t.wg,
			DbDao:              t.dbDao,
			CN:                 t.cn,
			ParserType:         tables.ParserTypeETH,
			PayTokenId:         tables.PayTokenIdETH,
			ContractPayTokenId: tables.PayTokenIdErc20USDT,
			ContractAddress:    tables.PayTokenIdErc20USDT.GetContractAddress(config.Cfg.Server.Net),
			CurrentBlockNumber: 0,
			ConcurrencyNum:     5,
			ConfirmNum:         2,
			Switch:             config.Cfg.Chain.Eth.Switch,
			AddrMap:            config.FormatAddrMap(tables.ParserTypeETH, config.Cfg.Chain.Eth.AddrMap),
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
	t.parserCommonMap[tables.ParserTypeBSC] = &parser_common.ParserCommon{
		PC: &parser_common.ParserCore{
			Ctx:                t.ctx,
			Wg:                 t.wg,
			DbDao:              t.dbDao,
			CN:                 t.cn,
			ParserType:         tables.ParserTypeBSC,
			PayTokenId:         tables.PayTokenIdBNB,
			ContractPayTokenId: tables.PayTokenIdBep20USDT,
			ContractAddress:    tables.PayTokenIdBep20USDT.GetContractAddress(config.Cfg.Server.Net),
			CurrentBlockNumber: 0,
			ConcurrencyNum:     10,
			ConfirmNum:         10,
			Switch:             config.Cfg.Chain.Bsc.Switch,
			AddrMap:            config.FormatAddrMap(tables.ParserTypeBSC, config.Cfg.Chain.Bsc.AddrMap),
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
	t.parserCommonMap[tables.ParserTypePOLYGON] = &parser_common.ParserCommon{
		PC: &parser_common.ParserCore{
			Ctx:                t.ctx,
			Wg:                 t.wg,
			DbDao:              t.dbDao,
			CN:                 t.cn,
			ParserType:         tables.ParserTypePOLYGON,
			PayTokenId:         tables.PayTokenIdPOL, //tables.PayTokenIdMATIC,
			CurrentBlockNumber: 0,
			ConcurrencyNum:     10,
			ConfirmNum:         10,
			Switch:             config.Cfg.Chain.Polygon.Switch,
			AddrMap:            config.FormatAddrMap(tables.ParserTypePOLYGON, config.Cfg.Chain.Polygon.AddrMap),
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
	contractAddress := tables.PayTokenIdTrc20USDT.GetContractAddress(config.Cfg.Server.Net)
	if contractAddress, err = common.TronBase58ToHex(contractAddress); err != nil {
		return fmt.Errorf("TronBase58ToHex err: %s", err.Error())
	}
	t.parserCommonMap[tables.ParserTypeTRON] = &parser_common.ParserCommon{
		PC: &parser_common.ParserCore{
			Ctx:                t.ctx,
			Wg:                 t.wg,
			DbDao:              t.dbDao,
			CN:                 t.cn,
			ParserType:         tables.ParserTypeTRON,
			PayTokenId:         tables.PayTokenIdTRX,
			ContractPayTokenId: tables.PayTokenIdTrc20USDT,
			ContractAddress:    contractAddress,
			CurrentBlockNumber: 0,
			ConcurrencyNum:     10,
			ConfirmNum:         10,
			Switch:             config.Cfg.Chain.Tron.Switch,
			AddrMap:            config.FormatAddrMap(tables.ParserTypeTRON, config.Cfg.Chain.Tron.AddrMap),
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
	t.parserCommonMap[tables.ParserTypeCKB] = &parser_common.ParserCommon{
		PC: &parser_common.ParserCore{
			Ctx:                t.ctx,
			Wg:                 t.wg,
			DbDao:              t.dbDao,
			CN:                 t.cn,
			ParserType:         tables.ParserTypeCKB,
			PayTokenId:         tables.PayTokenIdDAS,
			CurrentBlockNumber: 0,
			ConcurrencyNum:     10,
			ConfirmNum:         3,
			Switch:             config.Cfg.Chain.Ckb.Switch,
			AddrMap:            config.FormatAddrMap(tables.ParserTypeCKB, config.Cfg.Chain.Ckb.AddrMap),
		},
		PA: &parser_ckb.ParserCkb{
			Ctx:    t.ctx,
			Client: rpcClient,
		},
	}
	return nil
}

func (t *ToolParser) initParserDP() error {
	if !config.Cfg.Chain.DP.Switch {
		return nil
	}
	t.parserCommonMap[tables.ParserTypeDP] = &parser_common.ParserCommon{
		PC: &parser_common.ParserCore{
			Ctx:                t.ctx,
			Wg:                 t.wg,
			DbDao:              t.dbDao,
			CN:                 t.cn,
			ParserType:         tables.ParserTypeDP,
			PayTokenId:         tables.PayTokenIdDIDPoint,
			CurrentBlockNumber: 0,
			ConcurrencyNum:     10,
			ConfirmNum:         3,
			Switch:             config.Cfg.Chain.DP.Switch,
			AddrMap:            nil,
		},
		PA: &parser_dp.ParserDP{
			Ctx:     t.ctx,
			DasCore: t.dasCore,
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
	t.parserCommonMap[tables.ParserTypeDoge] = &parser_common.ParserCommon{
		PC: &parser_common.ParserCore{
			Ctx:                t.ctx,
			Wg:                 t.wg,
			DbDao:              t.dbDao,
			CN:                 t.cn,
			ParserType:         tables.ParserTypeDoge,
			PayTokenId:         tables.PayTokenIdDOGE,
			CurrentBlockNumber: 0,
			ConcurrencyNum:     3,
			ConfirmNum:         3,
			Switch:             config.Cfg.Chain.Doge.Switch,
			AddrMap:            config.Cfg.Chain.Doge.AddrMap,
		},
		PA: &parser_bitcoin.ParserBitcoin{NodeRpc: &nodeRpc},
	}
	return nil
}

func (t *ToolParser) RunParser() {
	for _, v := range t.parserCommonMap {
		if v.PC.Switch {
			go v.Parser()
		}
	}
}
