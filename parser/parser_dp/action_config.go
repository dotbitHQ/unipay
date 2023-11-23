package parser_dp

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"unipay/parser/parser_common"
)

func (p *ParserDP) ActionConfig(req FuncTransactionHandleReq, pc *parser_common.ParserCore) (resp FuncTransactionHandleResp) {
	if isCV, _, err := CurrentVersionTx(req.Tx, common.DasContractNameConfigCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		return
	}

	log.Info("ActionConfig:", req.TxHash)
	if err := p.DasCore.AsyncDasConfigCell(); err != nil {
		resp.Err = fmt.Errorf("AsyncDasConfigCell err: %s", err.Error())
		return
	}
	return
}
