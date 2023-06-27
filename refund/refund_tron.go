package refund

import (
	"encoding/hex"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"unipay/config"
	"unipay/tables"
)

func (t *ToolRefund) refundTron2(paymentAddress, private string, info tables.ViewRefundPaymentInfo) error {
	if !config.Cfg.Chain.Tron.Refund {
		return nil
	}
	amount := info.Amount
	orderId := info.OrderId
	toAddr := info.PayAddress
	payHash := info.PayHash
	payTokenId := info.PayTokenId
	fromHex := paymentAddress
	var err error

	var tx *api.TransactionExtention
	switch payTokenId {
	case tables.PayTokenIdTrc20USDT:
		//feeUSDT := decimal.NewFromInt(1e6)
		//if amount.Cmp(feeUSDT) != 1 {
		//	// NOTE fee more than refundAmount
		//	if err = t.DbDao.UpdateRefundStatusToRejected(payHash); err != nil {
		//		log.Error("UpdateRefundStatusToRejected err: ", err.Error(), payHash)
		//	}
		//	return nil
		//}

		contractHex := payTokenId.GetContractAddress(config.Cfg.Server.Net)
		if contractHex, err = common.TronBase58ToHex(contractHex); err != nil {
			return fmt.Errorf("TronBase58ToHex err: %s", err.Error())
		}

		tx, err = t.chainTron.TransferTrc20(contractHex, fromHex, toAddr, amount.IntPart(), 20*1e6)
		if err != nil {
			return fmt.Errorf("TransferTrc20 err: %s", err.Error())
		}
	case tables.PayTokenIdTRX:
		tx, err = t.chainTron.CreateTransaction(fromHex, toAddr, orderId, amount.IntPart())
		if err != nil {
			return fmt.Errorf("CreateTransaction err: %s", err.Error())
		}
	default:
		return fmt.Errorf("unknow pay token id[%s]", payTokenId)
	}

	if private != "" {
		err = t.chainTron.LocalSign(tx, private)
		if err != nil {
			return fmt.Errorf("AddSign err:%s", err.Error())
		}
	} else if t.remoteSignClient != nil {
		tx, err = t.remoteSignClient.SignTrxTx(fromHex, tx)
		if err != nil {
			return fmt.Errorf("SignTrxTx err: %s", err.Error())
		}
	} else {
		return fmt.Errorf("no signature method configured")
	}

	// send tx
	refundHash := hex.EncodeToString(tx.Txid)

	if err := t.DbDao.UpdateSinglePaymentToRefunded(payHash, refundHash, 0); err != nil {
		return fmt.Errorf("UpdateSinglePaymentToRefunded err: %s", err.Error())
	}
	if err = t.chainTron.SendTransaction(tx.Transaction); err != nil {
		if err = t.DbDao.UpdateSinglePaymentToUnRefunded(payHash); err != nil {
			log.Info("UpdateSinglePaymentToUnRefunded err: ", err.Error(), payHash)
			sendRefundNotify(info.Id, payTokenId, orderId, err.Error())
		}
		return fmt.Errorf("SendTx err: %s", err.Error())
	}

	// callback notice
	if err = t.addCallbackNotice2([]tables.ViewRefundPaymentInfo{info}); err != nil {
		log.Error("addCallbackNotice err:", err.Error())
	}

	return nil
}
