package refund

import (
	"context"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/unipay/dao"
	"github.com/dotbitHQ/unipay/parser"
	"sync"
)

type ToolRefund struct {
	Ctx        context.Context
	Wg         *sync.WaitGroup
	DbDao      *dao.DbDao
	DasCore    *core.DasCore
	ToolParser *parser.ToolParser
}

func (t *ToolRefund) RunRefund() {

}
