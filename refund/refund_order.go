package refund

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/chain/chain_evm"
	"unipay/config"
	"unipay/notify"
	"unipay/tables"
)

func (t *ToolRefund) doRefund() error {
	// get refund list
	list, err := t.DbDao.GetRefundListWithin3d()
	if err != nil {
		return fmt.Errorf("GetRefundListWithin3d err: %s", err.Error())
	}

	//
	var ckbList []tables.TablePaymentInfo
	var dogeList []tables.TablePaymentInfo
	var otherList []tables.TablePaymentInfo
	var stripeList []tables.TablePaymentInfo
	for i, v := range list {
		if v.PayHashStatus != tables.PayHashStatusConfirm && v.RefundStatus != tables.RefundStatusUnRefund {
			continue
		}
		switch v.PayTokenId {
		case tables.PayTokenIdCKB, tables.PayTokenIdDAS:
			ckbList = append(ckbList, list[i])
		case tables.PayTokenIdETH, tables.PayTokenIdBNB,
			tables.PayTokenIdMATIC, tables.PayTokenIdTRX, tables.PayTokenIdTrc20USDT,
			tables.PayTokenIdErc20USDT, tables.PayTokenIdBep20USDT:
			otherList = append(otherList, list[i])
		case tables.PayTokenIdStripeUSD:
			stripeList = append(stripeList, list[i])
		case tables.PayTokenIdDOGE:
			dogeList = append(dogeList, list[i])
		default:
			log.Warn("unknown pay token id[%s]", v.PayTokenId)
		}
	}

	// refund
	if err = t.doRefundCkb(ckbList); err != nil {
		log.Error("doRefundCkb err: ", err.Error())
		notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doRefundCKB", err.Error())
	}
	if err = t.doRefundDoge(dogeList); err != nil {
		log.Error("doRefundDoge err: ", err.Error())
		notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doRefundDoge", err.Error())
	}
	if err = t.doRefundOther(otherList); err != nil {
		log.Error("doRefundOther err: ", err.Error())
		notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doRefundOther", err.Error())
	}
	if err = t.doRefundStripe(stripeList); err != nil {
		log.Error("doRefundStripe err: ", err.Error())
		notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doRefundStripe", err.Error())
	}

	return nil
}

func (t *ToolRefund) doRefund2() error {
	// get refund list
	list, err := t.DbDao.GetViewRefundListWithin3d()
	if err != nil {
		return fmt.Errorf("GetViewRefundListWithin3d err: %s", err.Error())
	}
	//
	var refundMap = make(map[tables.ParserType]map[string][]tables.ViewRefundPaymentInfo)
	var stripeList []tables.ViewRefundPaymentInfo
	for i, v := range list {
		if v.PayHashStatus != tables.PayHashStatusConfirm && v.RefundStatus != tables.RefundStatusUnRefund {
			continue
		}
		var parserType tables.ParserType
		switch v.PayTokenId {
		case tables.PayTokenIdCKB, tables.PayTokenIdDAS:
			parserType = tables.ParserTypeCKB
		case tables.PayTokenIdETH, tables.PayTokenIdErc20USDT:
			parserType = tables.ParserTypeETH
		case tables.PayTokenIdBNB, tables.PayTokenIdBep20USDT:
			parserType = tables.ParserTypeBSC
		case tables.PayTokenIdMATIC:
			parserType = tables.ParserTypePOLYGON
		case tables.PayTokenIdTRX, tables.PayTokenIdTrc20USDT:
			parserType = tables.ParserTypeTRON
		case tables.PayTokenIdDOGE:
			parserType = tables.ParserTypeDoge
		case tables.PayTokenIdStripeUSD:
			stripeList = append(stripeList, list[i])
		default:
			log.Warn("unknown pay token id[%s]", v.PayTokenId)
			continue
		}
		if _, ok := refundMap[parserType]; !ok {
			refundMap[parserType] = make(map[string][]tables.ViewRefundPaymentInfo)
		}
		refundMap[parserType][v.PaymentAddress] = append(refundMap[parserType][v.PaymentAddress], list[i])
	}

	// do refund
	var parserTypeAddrMap = make(map[tables.ParserType]map[string]string)
	parserTypeAddrMap[tables.ParserTypeCKB] = config.Cfg.Chain.Ckb.AddrMap
	parserTypeAddrMap[tables.ParserTypeDoge] = config.Cfg.Chain.Doge.AddrMap
	parserTypeAddrMap[tables.ParserTypeTRON] = config.Cfg.Chain.Tron.AddrMap
	parserTypeAddrMap[tables.ParserTypeETH] = config.Cfg.Chain.Eth.AddrMap
	parserTypeAddrMap[tables.ParserTypeBSC] = config.Cfg.Chain.Bsc.AddrMap
	parserTypeAddrMap[tables.ParserTypePOLYGON] = config.Cfg.Chain.Polygon.AddrMap
	parserTypeEvmMap, err := t.getParserTypeEvmMap()
	if err != nil {
		return fmt.Errorf("getParserTypeEvmMap err: %s", err.Error())
	}

	for parserType, refundListMap := range refundMap {
		for paymentAddress, refundList := range refundListMap {
			addrMap := config.FormatAddrMap(parserType, parserTypeAddrMap[parserType])
			private, ok := addrMap[paymentAddress]
			if !ok {
				continue
			}
			switch parserType {
			case tables.ParserTypeCKB:
				err = t.doRefundCkb2(paymentAddress, private, refundList)
			case tables.ParserTypeDoge:
				err = t.doRefundDoge2(paymentAddress, private, refundList)
			case tables.ParserTypeTRON:
				for _, v := range refundList {
					if er := t.refundTron2(paymentAddress, private, v); err != nil {
						log.Error("refundTron2 err: ", parserType, paymentAddress, er.Error())
						notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "refundTron2", er.Error())
					}
				}
			case tables.ParserTypeETH, tables.ParserTypeBSC, tables.ParserTypePOLYGON:
				item := parserTypeEvmMap[parserType]
				for _, v := range refundList {
					if refundOK, er := t.refundEvm2(refundEvmParam2{
						info:        v,
						fromAddr:    paymentAddress,
						private:     private,
						addFee:      item.addFee,
						refund:      item.refund,
						chainEvm:    item.chainEvm,
						refundNonce: item.nonceMap[paymentAddress],
					}); er != nil {
						log.Error("refundEvm2 err:", err.Error(), v.PayTokenId, v.OrderId)
						sendRefundNotify(v.Id, v.PayTokenId, v.OrderId, err.Error())
					} else if refundOK {
						item.nonceMap[paymentAddress]++
						parserTypeEvmMap[parserType] = item
					}
				}
			}
			if err != nil {
				log.Error("doRefund2 err: ", parserType, paymentAddress, err.Error())
				notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doRefund2", err.Error())
			}
		}
	}
	// stripe
	if err = t.doRefundStripe2(stripeList); err != nil {
		log.Error("doRefundStripe2 err: ", err.Error())
		notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doRefundStripe2", err.Error())
	}

	return nil
}

type parserTypeEvm struct {
	addFee   float64
	refund   bool
	chainEvm *chain_evm.ChainEvm
	nonceMap map[string]uint64
}

func (t *ToolRefund) getParserTypeEvmMap() (map[tables.ParserType]parserTypeEvm, error) {
	var parserTypeEvmMap = make(map[tables.ParserType]parserTypeEvm)
	// eth
	parserTypeETH := parserTypeEvm{
		addFee:   config.Cfg.Chain.Eth.RefundAddFee,
		refund:   config.Cfg.Chain.Eth.Refund,
		chainEvm: t.chainEth,
		nonceMap: make(map[string]uint64),
	}
	for k, _ := range config.Cfg.Chain.Eth.AddrMap {
		nonce, err := t.chainEth.NonceAt(k)
		if err != nil {
			return nil, fmt.Errorf("NonceAt eth err: %s", err.Error())
		}
		parserTypeETH.nonceMap[k] = nonce
		payTokenIds := []tables.PayTokenId{tables.PayTokenIdETH, tables.PayTokenIdErc20USDT}
		nonceInfo, err := t.DbDao.GetRefundNonce(nonce, payTokenIds) // todo nonce
		if err != nil {
			return nil, fmt.Errorf("GetRefundNonce err: %s[%d][%v]", err.Error(), nonce, payTokenIds)
		} else if nonceInfo.Id > 0 {
			parserTypeETH.refund = false
		}
	}
	parserTypeEvmMap[tables.ParserTypeETH] = parserTypeETH

	// todo bsc
	parserTypeEvmMap[tables.ParserTypeBSC] = parserTypeEvm{
		addFee:   config.Cfg.Chain.Bsc.RefundAddFee,
		refund:   config.Cfg.Chain.Bsc.Refund,
		chainEvm: t.chainBsc,
		nonceMap: nil,
	}

	// todo polygon
	parserTypeEvmMap[tables.ParserTypePOLYGON] = parserTypeEvm{
		addFee:   config.Cfg.Chain.Polygon.RefundAddFee,
		refund:   config.Cfg.Chain.Polygon.Refund,
		chainEvm: t.chainPolygon,
		nonceMap: nil,
	}
	return parserTypeEvmMap, nil
}
