package parser_dp

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"unipay/dao"
	"unipay/parser/parser_common"
)

type FuncTransactionHandleReq struct {
	DbDao          *dao.DbDao
	Tx             *types.Transaction
	TxHash         string
	BlockNumber    uint64
	BlockTimestamp int64
	Action         common.DasAction
}

type FuncTransactionHandleResp struct {
	ActionName string
	Err        error
}

type FuncTransactionHandle func(FuncTransactionHandleReq, *parser_common.ParserCore) FuncTransactionHandleResp

func (p *ParserDP) registerTransactionHandle() {
	p.mapTransactionHandle = make(map[string]FuncTransactionHandle)
	p.mapTransactionHandle[common.DasActionConfig] = p.ActionConfig
	p.mapTransactionHandle[common.DasActionTransferDP] = p.ActionTransferDP
}

func CurrentVersionTx(tx *types.Transaction, name common.DasContractName) (bool, int, error) {
	contract, err := core.GetDasContractInfo(name)
	if err != nil {
		return false, -1, fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}

	idx := -1
	isCV := false
	for i, v := range tx.Outputs {
		if v.Type == nil {
			continue
		}
		if contract.IsSameTypeId(v.Type.CodeHash) {
			isCV = true
			idx = i
			break
		}
	}
	return isCV, idx, nil
}
