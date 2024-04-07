package refund

import (
	"fmt"
	"github.com/btcsuite/btcd/wire"
	"github.com/dotbitHQ/das-lib/bitcoin"
	"github.com/dotbitHQ/das-lib/remote_sign"
	"strings"
	"unipay/config"
	"unipay/notify"
	"unipay/tables"
)

func (t *ToolRefund) doRefundBTC(paymentAddress, private string, list []tables.ViewRefundPaymentInfo) error {
	if !config.Cfg.Chain.BTC.Refund {
		return fmt.Errorf("btc refund flag is false")
	}
	if t.chainBTC == nil {
		return fmt.Errorf("chainBTC client is nil")
	}
	if len(list) == 0 {
		return nil
	}

	var payHashList []string
	var addresses []string
	var values []int64
	var total int64
	for _, v := range list {
		payHashList = append(payHashList, v.PayHash)
		addresses = append(addresses, v.PayAddress)
		value := v.Amount.IntPart()
		total += value
		values = append(values, value)
	}
	if len(addresses) == 0 || len(values) == 0 {
		return nil
	}

	// get utxo
	_, utxos, err := bitcoin.GetUnspentOutputsBtc(paymentAddress, private, config.Cfg.Chain.BTC.UtxoApiUrl, config.Cfg.Chain.BTC.UtxoApiKey, total)
	if err != nil {
		return fmt.Errorf("bitcoin.GetUnspentOutputsBtc err: %s", err.Error())
	}

	// build tx
	tx, err := t.chainBTC.NewBTCTx(utxos, addresses, values, "")
	if err != nil {
		return fmt.Errorf("NewBTCTx err: %s", err.Error())
	}

	// sign
	var signTx *wire.MsgTx
	if private != "" {
		log.Info("doRefundBTC private")
		if _, err = t.chainBTC.LocalSignTxWithWitness(tx, utxos); err != nil {
			return fmt.Errorf("LocalSignTxWithWitness err: %s", err.Error())
		}
		signTx = tx
	} else if config.Cfg.Server.RemoteSignApiUrl != "" {
		log.Info("doRefundBTC remote sign")
		var witnessUTXOs []remote_sign.UTXO
		for _, v := range utxos {
			witnessUTXOs = append(witnessUTXOs, remote_sign.UTXO{
				Address: v.Address,
				Value:   v.Value,
			})
		}
		signTx, err = remote_sign.SignTxForBTC(config.Cfg.Server.RemoteSignApiUrl, paymentAddress, tx, witnessUTXOs)
		if err != nil {
			return fmt.Errorf("remote_sign.SignTxForBTC err: %s", err.Error())
		}
	} else {
		return fmt.Errorf("no signature configured")
	}

	// send tx
	refundHash := signTx.TxHash()
	if err := t.DbDao.UpdatePaymentListToRefunded(payHashList, refundHash.String()); err != nil {
		return fmt.Errorf("UpdatePaymentListToRefunded err: %s", err.Error())
	}
	if _, err = t.chainBTC.SendBTCTx(signTx); err != nil {
		if err = t.DbDao.UpdatePaymentListToUnRefunded(payHashList); err != nil {
			log.Info("UpdatePaymentListToUnRefunded err: ", err.Error(), payHashList)
			notify.SendLarkErrNotify("doRefundDoge", fmt.Sprintf("%s\n%s", strings.Join(payHashList, ","), err.Error()))
		}
		return fmt.Errorf("SendTx err: %s", err.Error())
	}

	// callback notice
	if err = t.addCallbackNotice(list); err != nil {
		log.Error("addCallbackNotice err:", err.Error())
	}
	return nil
}
