package example

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"testing"
	"unipay/http_svr/handle"
	"unipay/tables"
)

func TestOrderCreate(t *testing.T) {
	req := handle.ReqOrderCreate{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: common.CoinTypeEth,
				ChainId:  "11",
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

func TestOrderRefund(t *testing.T) {
	req := handle.ReqOrderRefund{
		BusinessId: "auto-sub-account",
		OrderId:    "0ba3ff32e5e585385073ad41305abf63",
	}
	url := fmt.Sprintf("%s%s", ApiUrl, "/order/refund")

	var data handle.RespOrderRefund
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println(toolib.JsonString(&data))
}

func TestOrderInfo(t *testing.T) {
	req := handle.ReqOrderInfo{
		BusinessId: "auto-sub-account",
		OrderId:    "0ba3ff32e5e585385073ad41305abf63",
	}
	url := fmt.Sprintf("%s%s", ApiUrl, "/order/info")

	var data handle.RespOrderInfo
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println(toolib.JsonString(&data))
}
