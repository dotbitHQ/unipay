package refund

import (
	"encoding/hex"
	"fmt"
	"github.com/dotbitHQ/das-lib/chain/chain_evm"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/remote_sign"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"strings"
	"unipay/config"
	"unipay/notify"
	"unipay/tables"
)

type refundEvmParam struct {
	info        tables.TablePaymentInfo
	fromAddr    string
	private     string
	addFee      float64
	refund      bool
	chainEvm    *chain_evm.ChainEvm
	refundNonce uint64
}

func (t *ToolRefund) refundEvm(p refundEvmParam) (ok bool, e error) {
	if !p.refund {
		return
	}

	data := []byte(p.info.OrderId)
	refundAmount := p.info.Amount
	addFee := p.addFee
	fromAddr := p.fromAddr
	refundNonce := p.refundNonce
	private := p.private
	toAddr := p.info.PayAddress
	payHash := p.info.PayHash

	// tx fee
	gasPrice, gasLimit, err := p.chainEvm.EstimateGas(fromAddr, toAddr, refundAmount, data, addFee)
	if err != nil {
		e = fmt.Errorf("EstimateGas err: %s", err.Error())
		return
	}
	fee := gasPrice.Mul(gasLimit)
	if refundAmount.Cmp(fee) == -1 {
		if err = t.DbDao.UpdateSinglePaymentToRefunded(payHash, "", 0); err != nil {
			log.Error("UpdateSinglePaymentToRefunded err: ", err.Error(), payHash)
		}
		return
	} else {
		refundAmount = refundAmount.Sub(fee)
	}
	log.Info("refundEvm:", p.info.OrderId, p.info.Amount, refundAmount, fee)

	// build tx
	tx, err := p.chainEvm.NewTransaction(fromAddr, toAddr, refundAmount, data, refundNonce, gasPrice, gasLimit)
	if err != nil {
		e = fmt.Errorf("NewTransaction err: %s", err.Error())
		return
	}
	if private != "" {
		tx, err = p.chainEvm.SignWithPrivateKey(private, tx)
		if err != nil {
			e = fmt.Errorf("SignWithPrivateKey err:%s", err.Error())
			return
		}
	} else if t.RemoteSignClient != nil {
		tx, err = t.RemoteSignClient.SignEvmTx(remote_sign.SignMethodEvm, fromAddr, tx)
		if err != nil {
			e = fmt.Errorf("SignEvmTx err: %s [%s]", err.Error(), p.info.PayTokenId)
			return
		}
	} else {
		e = fmt.Errorf("no signature method configured")
		return
	}

	// send tx
	refundHash := tx.Hash().Hex()

	if err := t.DbDao.UpdateSinglePaymentToRefunded(payHash, refundHash, refundNonce); err != nil {
		e = fmt.Errorf("UpdateSinglePaymentToRefunded err: %s", err.Error())
		return
	}
	if err = p.chainEvm.SendTransaction(tx); err != nil {
		if err = t.DbDao.UpdateSinglePaymentToUnRefunded(payHash); err != nil {
			log.Info("UpdateSinglePaymentToUnRefunded err: ", err.Error(), payHash)
			notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "refundEvm", fmt.Sprintf("%s\n%s", payHash, err.Error()))
		}
		e = fmt.Errorf("SendTx err: %s", err.Error())
		return
	}

	return true, nil
}

func (t *ToolRefund) refundTron(info tables.TablePaymentInfo) error {
	if !config.Cfg.Chain.Tron.Refund {
		return nil
	}
	amount := info.Amount
	orderId := info.OrderId
	toAddr := info.PayAddress
	payHash := info.PayHash
	payTokenId := info.PayTokenId
	fromHex := config.Cfg.Chain.Tron.Address
	if strings.HasPrefix(fromHex, common.TronBase58PreFix) {
		if fromData, err := address.Base58ToAddress(fromHex); err != nil {
			return fmt.Errorf("address.Base58ToAddress err: %s", err.Error())
		} else {
			fromHex = hex.EncodeToString(fromData)
		}
	}

	tx, err := t.ChainTron.CreateTransaction(fromHex, toAddr, orderId, amount.IntPart())
	if err != nil {
		return fmt.Errorf("CreateTransaction err: %s", err.Error())
	}

	if config.Cfg.Chain.Tron.Private != "" {
		tx, err = t.ChainTron.AddSign(tx.Transaction, config.Cfg.Chain.Tron.Private)
		if err != nil {
			return fmt.Errorf("AddSign err:%s", err.Error())
		}
	} else if t.RemoteSignClient != nil {
		tx, err = t.RemoteSignClient.SignTrxTx(fromHex, tx)
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
	if err = t.ChainTron.SendTransaction(tx.Transaction); err != nil {
		if err = t.DbDao.UpdateSinglePaymentToUnRefunded(payHash); err != nil {
			log.Info("UpdateSinglePaymentToUnRefunded err: ", err.Error(), payHash)
			sendRefundNotify(info.Id, payTokenId, orderId, err.Error())
		}
		return fmt.Errorf("SendTx err: %s", err.Error())
	}

	return nil
}

func (t *ToolRefund) doRefundOther(list []tables.TablePaymentInfo) error {
	if len(list) == 0 {
		return nil
	}
	// todo
	refundNonceETH := uint64(0)
	refundNonceBSC := uint64(0)
	refundNoncePolygon := uint64(0)
	for i, v := range list {
		switch v.PayTokenId {
		case tables.PayTokenIdETH:
			if ok, err := t.refundEvm(refundEvmParam{
				info:        list[i],
				fromAddr:    config.Cfg.Chain.Eth.Address,
				private:     config.Cfg.Chain.Eth.Private,
				addFee:      config.Cfg.Chain.Eth.RefundAddFee,
				refund:      config.Cfg.Chain.Eth.Refund,
				chainEvm:    t.ChainETH,
				refundNonce: refundNonceETH,
			}); err != nil {
				log.Error("refundEvm err:", err.Error(), v.PayTokenId, v.OrderId)
				sendRefundNotify(v.Id, v.PayTokenId, v.OrderId, err.Error())
			} else if ok {
				refundNonceETH++
			}
		case tables.PayTokenIdBNB:
			if ok, err := t.refundEvm(refundEvmParam{
				info:        list[i],
				fromAddr:    config.Cfg.Chain.Bsc.Address,
				private:     config.Cfg.Chain.Bsc.Private,
				addFee:      config.Cfg.Chain.Bsc.RefundAddFee,
				refund:      config.Cfg.Chain.Bsc.Refund,
				chainEvm:    t.ChainBSC,
				refundNonce: refundNonceBSC,
			}); err != nil {
				log.Error("refundEvm err:", err.Error(), v.PayTokenId, v.OrderId)
				sendRefundNotify(v.Id, v.PayTokenId, v.OrderId, err.Error())
			} else if ok {
				refundNonceBSC++
			}
		case tables.PayTokenIdMATIC:
			if ok, err := t.refundEvm(refundEvmParam{
				info:        list[i],
				fromAddr:    config.Cfg.Chain.Polygon.Address,
				private:     config.Cfg.Chain.Polygon.Private,
				addFee:      config.Cfg.Chain.Polygon.RefundAddFee,
				refund:      config.Cfg.Chain.Polygon.Refund,
				chainEvm:    t.ChainPolygon,
				refundNonce: refundNoncePolygon,
			}); err != nil {
				log.Error("refundEvm err:", err.Error(), v.PayTokenId, v.OrderId)
				sendRefundNotify(v.Id, v.PayTokenId, v.OrderId, err.Error())
			} else if ok {
				refundNoncePolygon++
			}
		case tables.PayTokenIdTRX:
			if err := t.refundTron(list[i]); err != nil {
				log.Error("refundTron err:", err.Error(), v.PayTokenId, v.OrderId)
				sendRefundNotify(v.Id, v.PayTokenId, v.OrderId, err.Error())
			}
		default:
			log.Error("unknown PayTokenId:", v.PayTokenId)
		}
	}
	return nil
}

func sendRefundNotify(id uint64, payTokenId tables.PayTokenId, orderId, err string) {
	msg := fmt.Sprintf("ID: %d\nPayTokenId: %s\nOrderId: %s\nErr: %s", id, payTokenId, orderId, err)
	notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "refund", msg)
}
