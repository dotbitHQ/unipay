package parser_dp

import (
	"encoding/hex"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/nervosnetwork/ckb-sdk-go/address"
	"github.com/shopspring/decimal"
	"math/big"
	"unipay/config"
	"unipay/parser/parser_common"
	"unipay/tables"
)

func (p *ParserDP) ActionTransferDP(req FuncTransactionHandleReq, pc *parser_common.ParserCore) (resp FuncTransactionHandleResp) {
	if isCV, _, err := CurrentVersionTx(req.Tx, common.DasContractNameDpCellType); err != nil {
		resp.Err = fmt.Errorf("CurrentVersionTx err: %s", err.Error())
		return
	} else if !isCV {
		return
	}
	log.Info("ActionTransferDP:", req.TxHash)
	parserType := pc.ParserType

	// check action
	//log.Info("req.Tx:", req.Tx.Hash.Hex(), toolib.JsonString(req.Tx))
	txOrderInfo, err := witness.DPOrderInfoFromTx(req.Tx)
	if err != nil {
		if err.Error() == witness.ErrNotExistDPOrderInfo.Error() {
			log.Warn("ActionTransferDP:", err.Error(), req.TxHash)
			return
		} else {
			resp.Err = fmt.Errorf("DPOrderInfoFromTx err: %s", err.Error())
			return
		}
	}
	actionMap := map[witness.DPAction]struct{}{
		witness.DPActionTransferCoupon: {},
		witness.DPActionTransferTLDID:  {},
		witness.DPActionTransferSLDID:  {},
	}
	if _, ok := actionMap[txOrderInfo.Action]; !ok {
		return
	}
	// txDPInfo
	dpInputs, err := p.DasCore.GetInputsDPInfo(req.Tx)
	if err != nil {
		resp.Err = fmt.Errorf("GetInputsDPInfo err: %s", err.Error())
		return
	}
	dpOutputs, err := p.DasCore.GetOutputsDPInfo(req.Tx)
	if err != nil {
		resp.Err = fmt.Errorf("GetOutputsDPInfo err: %s", err.Error())
		return
	}
	var txDPInfoOfSvr core.TxDPInfo
	for k, _ := range pc.AddrMap {
		log.Info("AddrMap:", k)
		addrHex, _, err := p.DasCore.Daf().ArgsToHex(common.Hex2Bytes(k))
		if err != nil {
			resp.Err = fmt.Errorf("ArgsToHex err: %s", err.Error())
			return
		}
		if item, ok := dpOutputs[hex.EncodeToString(addrHex.AddressPayload)]; ok {
			txDPInfoOfSvr = item
		}
	}
	if txDPInfoOfSvr.Payload == "" {
		resp.Err = fmt.Errorf("txDPInfoOfSvr.Payload is nil")
		return
	}
	//
	var txDPInfoOfUser core.TxDPInfo
	for k, v := range dpInputs {
		if item, ok := dpOutputs[k]; ok {
			v.AmountDP -= item.AmountDP
			txDPInfoOfUser = v
			break
		}
	}
	if txDPInfoOfUser.Payload == "" {
		resp.Err = fmt.Errorf("txDPInfo.Payload is nil: %s", req.TxHash)
		return
	}
	if txDPInfoOfUser.AmountDP != txDPInfoOfSvr.AmountDP {
		log.Warn("txDPInfoOfUser.AmountDP != txDPInfoOfSvr.AmountDP:", txDPInfoOfUser.AmountDP, txDPInfoOfSvr.AmountDP)
		resp.Err = fmt.Errorf("txDPInfoOfUser.AmountDP != txDPInfoOfSvr.AmountDP: %s", req.TxHash)
		return
	}
	userLock, _, err := p.DasCore.Daf().HexToScript(core.DasAddressHex{
		DasAlgorithmId:    txDPInfoOfUser.AlgId,
		DasSubAlgorithmId: txDPInfoOfUser.SubAlgId,
		AddressHex:        txDPInfoOfUser.Payload,
		IsMulti:           false,
		ChainType:         0,
	})
	if err != nil {
		resp.Err = fmt.Errorf("HexToScript err: %s", err.Error())
		return
	}
	mode := address.Mainnet
	if config.Cfg.Server.Net != common.DasNetTypeMainNet {
		mode = address.Testnet
	}
	fromAddr, err := common.ConvertScriptToAddress(mode, userLock)
	if err != nil {
		resp.Err = fmt.Errorf("common.ConvertScriptToAddress err:%s", err.Error())
		return
	}
	//
	orderId := txOrderInfo.OrderId
	order, err := pc.DbDao.GetOrderInfoByOrderId(orderId)
	if err != nil {
		resp.Err = fmt.Errorf("GetOrderInfoByOrderId err: %s", err.Error())
		return
	} else if order.Id == 0 {
		log.Warn("ActionTransferDP: order not exist:", parserType, orderId, req.TxHash)
		return
	}
	amountOrder := order.Amount.BigInt().Uint64()
	log.Info("ActionTransferDP:", txDPInfoOfUser.Payload, txDPInfoOfUser.AmountDP, amountOrder)
	if order.PayTokenId != tables.PayTokenIdDIDPoint {
		log.Warn("order pay token id not match", order.OrderId)
		return
	}
	if txDPInfoOfUser.AmountDP < amountOrder {
		amountDP := decimal.NewFromBigInt(new(big.Int).SetUint64(txDPInfoOfUser.AmountDP), 0)
		pc.CreatePaymentForMismatch(order.OrderId, req.TxHash, fromAddr, amountDP, order.PayTokenId)
		return
	}

	// change the status to confirm
	if err = pc.DoPayment(order, req.TxHash, fromAddr, parserType.ToAlgorithmId()); err != nil {
		resp.Err = fmt.Errorf("pc.DoPayment err: %s", err.Error())
		return
	}

	return
}
