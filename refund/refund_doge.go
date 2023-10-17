package refund

import (
	"fmt"
	"github.com/btcsuite/btcd/wire"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/remote_sign"
	"strings"
	"time"
	"unipay/config"
	"unipay/notify"
	"unipay/tables"
)

func (t *ToolRefund) doRefundDoge(paymentAddress, private string, list []tables.ViewRefundPaymentInfo) error {
	if !config.Cfg.Chain.Doge.Refund {
		return fmt.Errorf("doge refund flag is false")
	}
	if t.chainDoge == nil {
		return fmt.Errorf("chainDoge client is nil")
	}
	if len(list) == 0 {
		return nil
	}
	var payHashList []string
	var addresses []string
	var values []int64
	var total int64
	for _, v := range list {
		dogeAddr, err := common.Base58CheckEncode(v.PayAddress, common.DogeCoinBase58Version)
		if err != nil {
			return fmt.Errorf("Base58CheckEncode err: %s", err.Error())
		}
		// NOTE check amount
		payHashList = append(payHashList, v.PayHash)
		addresses = append(addresses, dogeAddr)
		value := v.Amount.IntPart()
		total += value
		values = append(values, value)
	}
	if len(addresses) == 0 || len(values) == 0 {
		return nil
	}

	// get utxo
	_, uos, err := t.chainDoge.GetUnspentOutputsDoge(paymentAddress, private, total)
	if err != nil {
		return fmt.Errorf("GetUnspentOutputsDoge err: %s", err.Error())
	}

	// build tx
	tx, err := t.chainDoge.NewTx(uos, addresses, values, "")
	if err != nil {
		return fmt.Errorf("NewTx err: %s", err.Error())
	}

	// sign
	var signTx *wire.MsgTx
	if private != "" {
		log.Info("doRefundDoge private")
		if _, err = t.chainDoge.LocalSignTx(tx, uos); err != nil {
			return fmt.Errorf("LocalSignTx err: %s", err.Error())
		}
		signTx = tx
	} else if config.Cfg.Server.RemoteSignApiUrl != "" {
		log.Info("doRefundDoge remote sign")
		signTx, err = remote_sign.SignTxForDOGE(config.Cfg.Server.RemoteSignApiUrl, paymentAddress, tx)
		if err != nil {
			return fmt.Errorf("remote_sign.SignTxForDOGE err: %s", err.Error())
		}
	} else {
		return fmt.Errorf("no signature configured")
	}
	//else if t.chainDoge.RemoteSignClient != nil {
	//	if signTx, err = t.chainDoge.RemoteSignTx(bitcoin.RemoteSignMethodDogeTx, tx, uos); err != nil {
	//		return fmt.Errorf("RemoteSignTx err: %s", err.Error())
	//	}
	//}

	// send tx
	refundHash := signTx.TxHash()
	if err := t.DbDao.UpdatePaymentListToRefunded(payHashList, refundHash.String()); err != nil {
		return fmt.Errorf("UpdatePaymentListToRefunded err: %s", err.Error())
	}
	if _, err = t.chainDoge.SendTx(signTx); err != nil {
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

func (t *ToolRefund) addCallbackNotice(list []tables.ViewRefundPaymentInfo) error {
	var noticeList []tables.TableNoticeInfo
	for _, v := range list {
		notice := tables.TableNoticeInfo{
			EventType:    tables.EventTypeOrderRefund,
			PayHash:      v.PayHash,
			NoticeCount:  0,
			NoticeStatus: tables.NoticeStatusDefault,
			Timestamp:    time.Now().UnixMilli(),
		}
		notice.InitNoticeId()
		noticeList = append(noticeList, notice)
	}

	if err := t.DbDao.CreateNoticeList(noticeList); err != nil {
		return fmt.Errorf("CreateNoticeList erR: %s", err.Error())
	}
	return nil
}
