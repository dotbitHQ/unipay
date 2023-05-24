package refund

import (
	"encoding/hex"
	"fmt"
	"github.com/dotbitHQ/das-lib/chain/chain_evm"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/remote_sign"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/shopspring/decimal"
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
	gasPrice, gasLimit, fee := decimal.Zero, decimal.Zero, decimal.Zero
	var err error

	switch p.info.PayTokenId {
	case tables.PayTokenIdErc20USDT, tables.PayTokenIdBep20USDT:
		feeUSDT := decimal.NewFromInt(5 * 1e6)
		if p.info.PayTokenId == tables.PayTokenIdErc20USDT {
			feeUSDT = decimal.NewFromInt(5 * 1e6)
		} else if p.info.PayTokenId == tables.PayTokenIdBep20USDT {
			feeUSDT = decimal.NewFromInt(1 * 1e6)
		} else {
			feeUSDT = decimal.NewFromInt(5 * 1e6)
		}
		if refundAmount.Cmp(feeUSDT) != 1 {
			// NOTE fee more than refundAmount
			if err = t.DbDao.UpdateRefundStatusToRejected(payHash); err != nil {
				log.Error("UpdateRefundStatusToRejected err: ", err.Error(), payHash)
			}
			return
		}
		refundAmount = refundAmount.Sub(feeUSDT)

		data, err = chain_evm.PackMessage("transfer", ethcommon.HexToAddress(toAddr), refundAmount.Coefficient())
		if err != nil {
			e = fmt.Errorf("chain_evm.PackMessage err: %s", err.Error())
			return
		}
		contract := p.info.PayTokenId.GetContractAddress(config.Cfg.Server.Net)
		gasPrice, gasLimit, err = p.chainEvm.EstimateGas(fromAddr, contract, decimal.Zero, data, addFee)
		if err != nil {
			e = fmt.Errorf("p.chainEvm.EstimateGas err: %s", err.Error())
			return
		}
		fee = gasPrice.Mul(gasLimit)
		toAddr = contract
		refundAmount = decimal.Zero
		return
	default:
		// tx fee
		gasPrice, gasLimit, err = p.chainEvm.EstimateGas(fromAddr, toAddr, refundAmount, data, addFee)
		if err != nil {
			e = fmt.Errorf("EstimateGas err: %s", err.Error())
			return
		}
		fee = gasPrice.Mul(gasLimit)

		// NOTE fee more than refundAmount
		if refundAmount.Cmp(fee) != 1 {
			if err = t.DbDao.UpdateRefundStatusToRejected(payHash); err != nil {
				log.Error("UpdateRefundStatusToRejected err: ", err.Error(), payHash)
			}
			return
		} else {
			refundAmount = refundAmount.Sub(fee)
		}
	}
	log.Info("refundEvm:", p.info.OrderId, p.info.PayTokenId, p.info.Amount, refundAmount, fee)

	// build tx
	tx, err := p.chainEvm.NewTransaction(fromAddr, toAddr, refundAmount, data, refundNonce, gasPrice, gasLimit)
	if err != nil {
		e = fmt.Errorf("NewTransaction err: %s", err.Error())
		return
	}
	if private != "" {
		log.Info("refundEvm private")
		tx, err = p.chainEvm.SignWithPrivateKey(private, tx)
		if err != nil {
			e = fmt.Errorf("SignWithPrivateKey err:%s", err.Error())
			return
		}
	} else if t.remoteSignClient != nil {
		log.Info("refundEvm remoteSignClient")
		tx, err = t.remoteSignClient.SignEvmTx(remote_sign.SignMethodEvm, fromAddr, tx)
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
		e = fmt.Errorf("SendTx err: %s", err.Error())
		if err = t.DbDao.UpdateSinglePaymentToUnRefunded(payHash); err != nil {
			log.Info("UpdateSinglePaymentToUnRefunded err: ", err.Error(), payHash)
			notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "refundEvm", fmt.Sprintf("%s\n%s", payHash, err.Error()))
		}
		return
	}

	// callback notice
	if err = t.addCallbackNotice([]tables.TablePaymentInfo{p.info}); err != nil {
		log.Error("addCallbackNotice err:", err.Error())
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

	tx, err := t.chainTron.CreateTransaction(fromHex, toAddr, orderId, amount.IntPart())
	if err != nil {
		return fmt.Errorf("CreateTransaction err: %s", err.Error())
	}

	if config.Cfg.Chain.Tron.Private != "" {
		err = t.chainTron.LocalSign(tx, config.Cfg.Chain.Tron.Private)
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
	if err = t.addCallbackNotice([]tables.TablePaymentInfo{info}); err != nil {
		log.Error("addCallbackNotice err:", err.Error())
	}

	return nil
}

func (t *ToolRefund) doRefundOther(list []tables.TablePaymentInfo) error {
	if len(list) == 0 {
		return nil
	}
	// get nonce
	refundEth, refundBsc, refundPolygon := config.Cfg.Chain.Eth.Refund, config.Cfg.Chain.Bsc.Refund, config.Cfg.Chain.Polygon.Refund
	refundNonceETH, refundNonceBSC, refundNoncePolygon := uint64(0), uint64(0), uint64(0)
	if refundEth && t.chainEth != nil {
		nonce, err := t.chainEth.NonceAt(config.Cfg.Chain.Eth.Address)
		if err != nil {
			return fmt.Errorf("NonceAt eth err: %s", err.Error())
		}
		refundNonceETH = nonce
		payTokenIds := []tables.PayTokenId{tables.PayTokenIdETH}
		nonceInfo, err := t.DbDao.GetRefundNonce(refundNonceETH, payTokenIds)
		if err != nil {
			return fmt.Errorf("GetRefundNonce err: %s[%d][%v]", err.Error(), refundNonceETH, payTokenIds)
		} else if nonceInfo.Id > 0 {
			refundEth = false
		}
	}
	if refundBsc && t.chainBsc != nil {
		nonce, err := t.chainBsc.NonceAt(config.Cfg.Chain.Bsc.Address)
		if err != nil {
			return fmt.Errorf("NonceAt bsc err: %s", err.Error())
		}
		refundNonceBSC = nonce
		payTokenIds := []tables.PayTokenId{tables.PayTokenIdBNB}
		nonceInfo, err := t.DbDao.GetRefundNonce(refundNonceBSC, payTokenIds)
		if err != nil {
			return fmt.Errorf("GetRefundNonce err: %s[%d][%v]", err.Error(), refundNonceBSC, payTokenIds)
		} else if nonceInfo.Id > 0 {
			refundEth = false
		}
	}
	if refundPolygon && t.chainPolygon != nil {
		nonce, err := t.chainPolygon.NonceAt(config.Cfg.Chain.Polygon.Address)
		if err != nil {
			return fmt.Errorf("NonceAt polygon err: %s", err.Error())
		}
		refundNoncePolygon = nonce
		payTokenIds := []tables.PayTokenId{tables.PayTokenIdMATIC}
		nonceInfo, err := t.DbDao.GetRefundNonce(refundNoncePolygon, payTokenIds)
		if err != nil {
			return fmt.Errorf("GetRefundNonce err: %s[%d][%v]", err.Error(), refundNoncePolygon, payTokenIds)
		} else if nonceInfo.Id > 0 {
			refundEth = false
		}
	}

	// refund
	for i, v := range list {
		switch v.PayTokenId {
		case tables.PayTokenIdETH, tables.PayTokenIdErc20USDT:
			if ok, err := t.refundEvm(refundEvmParam{
				info:        list[i],
				fromAddr:    config.Cfg.Chain.Eth.Address,
				private:     config.Cfg.Chain.Eth.Private,
				addFee:      config.Cfg.Chain.Eth.RefundAddFee,
				refund:      refundEth,
				chainEvm:    t.chainEth,
				refundNonce: refundNonceETH,
			}); err != nil {
				log.Error("refundEvm err:", err.Error(), v.PayTokenId, v.OrderId)
				sendRefundNotify(v.Id, v.PayTokenId, v.OrderId, err.Error())
			} else if ok {
				refundNonceETH++
			}
		case tables.PayTokenIdBNB, tables.PayTokenIdBep20USDT:
			if ok, err := t.refundEvm(refundEvmParam{
				info:        list[i],
				fromAddr:    config.Cfg.Chain.Bsc.Address,
				private:     config.Cfg.Chain.Bsc.Private,
				addFee:      config.Cfg.Chain.Bsc.RefundAddFee,
				refund:      refundBsc,
				chainEvm:    t.chainBsc,
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
				refund:      refundPolygon,
				chainEvm:    t.chainPolygon,
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
