package example

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/unipay/http_svr/handle"
	"github.com/dotbitHQ/unipay/tables"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"testing"
)

func TestOrderCreate(t *testing.T) {
	req := handle.ReqOrderCreate{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: common.CoinTypeEth,
				ChainId:  "",
				Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
			},
		},
		BusinessId: "auto-sub-account",
		Amount:     decimal.NewFromInt(1e18),
		PayTokenId: tables.PayTokenIdETH,
	}
	url := fmt.Sprintf("%s%s", ApiUrl, "/order/create")

	var data handle.RespOrderCreate
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println(toolib.JsonString(&data))
}
