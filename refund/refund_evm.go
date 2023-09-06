package refund

import (
	"context"
	"fmt"
	"github.com/dotbitHQ/das-lib/chain/chain_evm"
	"github.com/dotbitHQ/das-lib/remote_sign"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"unipay/config"
	"unipay/notify"
	"unipay/tables"
)

type refundEvmParam struct {
	info        tables.ViewRefundPaymentInfo
	fromAddr    string
	private     string
	addFee      float64
	refund      bool
	chainEvm    *chain_evm.ChainEvm
	refundNonce uint64
}

func (t *ToolRefund) refundEvm(p refundEvmParam) (ok bool, e error) {
	if !p.refund {
		e = fmt.Errorf("evm refund flag is false")
		return
	}
	if p.chainEvm == nil {
		e = fmt.Errorf("chainEvm client is nil")
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

	log.Warn("refundEvm:", p.info.OrderId, p.info.PayTokenId, p.info.Amount)

	switch p.info.PayTokenId {
	case tables.PayTokenIdErc20USDT, tables.PayTokenIdBep20USDT:
		feeUSDT := decimal.NewFromInt(5 * 1e6)
		if p.info.PayTokenId == tables.PayTokenIdBep20USDT {
			feeUSDT = decimal.NewFromInt(1 * 1e6)
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
	} else if config.Cfg.Server.RemoteSignApiUrl != "" {
		log.Info("refundEvm remote sign")
		chainID, err := p.chainEvm.Client.ChainID(context.Background())
		if err != nil {
			e = fmt.Errorf("p.chainEvm.Client.ChainID err: %s", err.Error())
			return
		}
		tx, err = remote_sign.SignTxForEVM(config.Cfg.Server.RemoteSignApiUrl, fromAddr, chainID.Int64(), tx)
		if err != nil {
			e = fmt.Errorf("remote_sign.SignTxForEVM err: %s", err.Error())
			return
		}
	} else {
		e = fmt.Errorf("no signature method configured")
		return
	}
	//else if t.remoteSignClient != nil {
	//	log.Info("refundEvm remoteSignClient")
	//	tx, err = t.remoteSignClient.SignEvmTx(remote_sign.SignMethodEvm, fromAddr, tx)
	//	if err != nil {
	//		e = fmt.Errorf("SignEvmTx err: %s [%s]", err.Error(), p.info.PayTokenId)
	//		return
	//	}
	//}

	// send tx
	refundHash := tx.Hash().Hex()

	if err := t.DbDao.UpdateSinglePaymentToRefunded2(payHash, refundHash, fromAddr, refundNonce); err != nil {
		e = fmt.Errorf("UpdateSinglePaymentToRefunded err: %s", err.Error())
		return
	}

	if err = p.chainEvm.SendTransaction(tx); err != nil {
		e = fmt.Errorf("SendTx err: %s", err.Error())
		if err = t.DbDao.UpdateSinglePaymentToUnRefunded(payHash); err != nil {
			log.Info("UpdateSinglePaymentToUnRefunded err: ", err.Error(), payHash)
			notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "UpdateSinglePaymentToUnRefunded", fmt.Sprintf("%s\n%s", payHash, err.Error()))
		}
		return
	}

	// callback notice
	if err = t.addCallbackNotice([]tables.ViewRefundPaymentInfo{p.info}); err != nil {
		log.Error("addCallbackNotice err:", err.Error())
	}

	return true, nil
}
