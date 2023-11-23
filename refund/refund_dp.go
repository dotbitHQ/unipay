package refund

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/remote_sign"
	"github.com/dotbitHQ/das-lib/sign"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/nervosnetwork/ckb-sdk-go/address"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"unipay/config"
	"unipay/notify"
	"unipay/tables"
)

func (t *ToolRefund) doRefundDP(list []tables.ViewRefundPaymentInfo) error {
	if !config.Cfg.Chain.DP.Refund {
		return fmt.Errorf("dp refund flag is false")
	}
	if len(list) == 0 {
		return nil
	}
	log.Info("doRefundDP:", len(list))
	chainId := int64(5)
	if config.Cfg.Server.Net == common.DasNetTypeMainNet {
		chainId = 1
	}
	fromLock, err := address.Parse(config.Cfg.Chain.DP.TransferWhitelist)
	if err != nil {
		return fmt.Errorf("address.Parse err: %s", err.Error())
	}
	fromAddr, _, err := t.DasCore.Daf().ScriptToHex(fromLock.Script)
	if err != nil {
		return fmt.Errorf("ScriptToHex err: %s", err.Error())
	}
	refundUrl := fmt.Sprintf("%s/v1/dp/refund", config.Cfg.Chain.DP.RefundUrl)
	sendTxUrl := fmt.Sprintf("%s/v1/tx/send", config.Cfg.Chain.DP.RefundUrl)
	for i, v := range list {

		req := ReqRefundDP{
			BusinessId:    v.BusinessId,
			OrderId:       v.OrderId,
			PayHash:       v.PayHash,
			RefundAddress: v.PayAddress,
			RefundAmount:  v.Amount,
		}
		var data RespRefundDP
		resp, err := http_api.SendReqV2(refundUrl, &req, &data)
		if err != nil {
			return fmt.Errorf("http_api.SendReqV2 err: %s", err.Error())
		}
		if resp.ErrNo != http_api.ApiCodeSuccess {
			return fmt.Errorf("req failed: [%d]%s", resp.ErrNo, resp.ErrMsg)
		}
		log.Info("RespRefundDP:", toolib.JsonString(&data))
		// sign
		for index, s := range data.SignList {
			if s.SignType != common.DasAlgorithmIdEth712 {
				continue
			}
			if config.Cfg.Chain.DP.TransferWhitelistPrivate != "" {
				sig, err := sign.DoEIP712Sign(chainId, s.SignMsg, config.Cfg.Chain.DP.TransferWhitelistPrivate, data.MMJson)
				if err != nil {
					return fmt.Errorf("DoEIP712Sign err: %s", err.Error())
				}
				data.SignList[index].SignMsg = sig
			} else if config.Cfg.Server.RemoteSignApiUrl != "" {
				sig, err := remote_sign.SignTxFor712(config.Cfg.Server.RemoteSignApiUrl, fromAddr.AddressHex, s.SignMsg, chainId, data.MMJson)
				if err != nil {
					return fmt.Errorf("SignTxFor712 err: %s", err.Error())
				}
				data.SignList[index].SignMsg = sig
			} else {
				return fmt.Errorf("no support remote sign")
			}
		}
		//
		if err := t.DbDao.UpdateSinglePaymentToRefunded(v.PayHash, data.Hash, 0); err != nil {
			return fmt.Errorf("UpdateSinglePaymentToRefunded err: %s", err.Error())
		}
		req2 := ReqTxSendDP{
			SignKey:  data.SignKey,
			SignList: data.SignList,
		}
		var data2 RespTxSendDP
		resp2, err := http_api.SendReqV2(sendTxUrl, &req2, &data2)
		if err != nil {
			if er := t.DbDao.UpdateSinglePaymentToUnRefunded(v.PayHash); er != nil {
				log.Info("UpdateSinglePaymentToUnRefunded err: ", er.Error(), v.PayHash)
				notify.SendLarkErrNotify("UpdateSinglePaymentToUnRefunded", fmt.Sprintf("%s\n%s", v.PayHash, er.Error()))
			}
			return fmt.Errorf("http_api.SendReqV2 err: %s", err.Error())
		}
		if resp2.ErrNo != http_api.ApiCodeSuccess {
			if er := t.DbDao.UpdateSinglePaymentToUnRefunded(v.PayAddress); er != nil {
				log.Info("UpdateSinglePaymentToUnRefunded err: ", er.Error(), v.PayHash)
				notify.SendLarkErrNotify("UpdateSinglePaymentToUnRefunded", fmt.Sprintf("%s\n%s", v.PayHash, er.Error()))
			}
			return fmt.Errorf("req failed: [%d]%s", resp2.ErrNo, resp2.ErrMsg)
		}
		// callback notice
		if err = t.addCallbackNotice([]tables.ViewRefundPaymentInfo{list[i]}); err != nil {
			log.Error("addCallbackNotice err:", err.Error())
		}
	}

	return nil
}

type ReqRefundDP struct {
	BusinessId    string          `json:"business_id"`
	OrderId       string          `json:"order_id"`
	PayHash       string          `json:"pay_hash"`
	RefundAddress string          `json:"refund_address"`
	RefundAmount  decimal.Decimal `json:"refund_amount"`
}

type RespRefundDP struct {
	Action   common.DasAction     `json:"action"`
	SignKey  string               `json:"sign_key" binding:"required"`
	SignList []txbuilder.SignData `json:"sign_list,omitempty"` // sign list
	MMJson   *common.MMJsonObj    `json:"mm_json,omitempty"`   // 712 mmjson
	Hash     string               `json:"hash"`
}

type ReqTxSendDP struct {
	SignKey  string               `json:"sign_key"`
	SignList []txbuilder.SignData `json:"sign_list"`
}

type RespTxSendDP struct {
	Hash string `json:"hash"`
}
