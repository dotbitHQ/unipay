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
	BusinessIdDasRegisterSvr = "das-register-svr"
	cta                      = core.ChainTypeAddress{
		Type: "blockchain",
		KeyInfo: core.KeyInfo{
			CoinType: common.CoinTypeTrx,
			ChainId:  "",
			//Key:      "0x15a33588908cF8Edb27D1AbE3852Bf287Abd3891",
			//Key: "0xD43B906Be6FbfFFFF60977A0d75EC93696e01dC7",
			//Key: "TQoLh9evwUmZKxpD1uhFttsZk3EBs8BksV",
			Key: "TFUg8zKThCj23acDSwsVjQrBVRywMMQGP1",
		},
	}
)

func TestOrderCreate(t *testing.T) {
	req := handle.ReqOrderCreate{
		ChainTypeAddress: cta,
		//BusinessId:       BusinessIdAutoSubAccount,
		BusinessId: BusinessIdDasRegisterSvr,
		Amount:     decimal.NewFromInt(4 * 1e6),
		PayTokenId: tables.PayTokenIdTrc20USDT,
		//PaymentAddress: "0xD43B906Be6FbfFFFF60977A0d75EC93696e01dC7",
		//PaymentAddress: "0x15a33588908cF8Edb27D1AbE3852Bf287Abd3891",
		//PaymentAddress: "TFUg8zKThCj23acDSwsVjQrBVRywMMQGP1",
		PaymentAddress: "TQoLh9evwUmZKxpD1uhFttsZk3EBs8BksV",
	}
	url := fmt.Sprintf("%s%s", ApiUrl, "/order/create")

	//fmt.Printf("curl -X POST %s -d'%s'\n", url, toolib.JsonString(req))

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

	fmt.Printf("curl -X POST %s -d'%s'", url, toolib.JsonString(&req))

	//var data handle.RespOrderRefund
	//if err := http_api.SendReq(url, req, &data); err != nil {
	//	t.Fatal(err)
	//}
	//fmt.Println(toolib.JsonString(&data))
}

func TestPaymentInfo(t *testing.T) {
	req := handle.ReqPaymentInfo{
		BusinessId:  "auto-sub-account",
		OrderIdList: []string{},
		PayHashList: []string{},
	}
	url := fmt.Sprintf("%s%s", ApiUrl, "/order/info")

	var data handle.RespPaymentInfo
	if err := http_api.SendReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
	fmt.Println(toolib.JsonString(&data))
}
