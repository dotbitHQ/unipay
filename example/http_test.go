package example

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"testing"
	"unipay/http_svr/handle"
	"unipay/tables"
)

var (
	BusinessIdAutoSubAccount = "auto-sub-account"
	cta                      = core.ChainTypeAddress{
		Type: "blockchain",
		KeyInfo: core.KeyInfo{
			CoinType: common.CoinTypeDogeCoin,
			ChainId:  "",
			Key:      "DQaRQ9s28U7EogPcDZudwZc4wD1NucZr2g",
		},
	}
)

func TestOrderCreate(t *testing.T) {
	req := handle.ReqOrderCreate{
		ChainTypeAddress: cta,
		BusinessId:       BusinessIdAutoSubAccount,
		Amount:           decimal.NewFromInt(2e8),
		PayTokenId:       tables.PayTokenIdDOGE,
	}
	url := fmt.Sprintf("%s%s", ApiUrl, "/order/create")

	var data handle.RespOrderCreate
	if err := http_api.SendReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println(toolib.JsonString(&data))
}

func TestOrderRefund(t *testing.T) {
	req := handle.ReqOrderRefund{
		BusinessId: "auto-sub-account",
		RefundList: []handle.RefundInfo{{
			OrderId: "",
			PayHash: "",
		}},
	}
	url := fmt.Sprintf("%s%s", ApiUrl, "/order/refund")

	var data handle.RespOrderRefund
	if err := http_api.SendReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println(toolib.JsonString(&data))
}

func TestOrderInfo(t *testing.T) {
	req := handle.ReqOrderInfo{
		BusinessId:  "auto-sub-account",
		OrderIdList: []string{},
		PayHashList: []string{},
	}
	url := fmt.Sprintf("%s%s", ApiUrl, "/order/info")

	var data handle.RespOrderInfo
	if err := http_api.SendReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println(toolib.JsonString(&data))
}
