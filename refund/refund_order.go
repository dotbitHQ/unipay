package refund

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/chain/chain_evm"
	"strings"
	"unipay/config"
	"unipay/notify"
	"unipay/tables"
)

func (t *ToolRefund) doRefund() error {
	// get refund list
	list, err := t.DbDao.GetViewRefundListWithin3d()
	if err != nil {
		return fmt.Errorf("GetViewRefundListWithin3d err: %s", err.Error())
	}
	//
	var refundMap = make(map[tables.ParserType]map[string][]tables.ViewRefundPaymentInfo)
	var stripeList []tables.ViewRefundPaymentInfo
	var dpList []tables.ViewRefundPaymentInfo
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
		case tables.PayTokenIdDIDPoint:
			dpList = append(dpList, list[i])
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
				err = t.doRefundCkb(paymentAddress, private, refundList)
			case tables.ParserTypeDoge:
				err = t.doRefundDoge(paymentAddress, private, refundList)
			case tables.ParserTypeTRON:
				for _, v := range refundList {
					if er := t.refundTron(paymentAddress, private, v); er != nil {
						log.Error("refundTron err: ", parserType, paymentAddress, er.Error())
						sendRefundNotify(v.Id, v.PayTokenId, v.OrderId, er.Error())
					}
				}
			case tables.ParserTypeETH, tables.ParserTypeBSC, tables.ParserTypePOLYGON:
				item := parserTypeEvmMap[parserType]
				for _, v := range refundList {
					if refundOK, er := t.refundEvm(refundEvmParam{
						info:        v,
						fromAddr:    paymentAddress,
						private:     private,
						addFee:      item.addFee,
						refund:      item.refund,
						chainEvm:    item.chainEvm,
						refundNonce: item.nonceMap[paymentAddress],
					}); er != nil {
						log.Error("refundEvm err:", er.Error(), v.PayTokenId, v.OrderId)
						sendRefundNotify(v.Id, v.PayTokenId, v.OrderId, er.Error())
					} else if refundOK {
						item.nonceMap[paymentAddress]++
						parserTypeEvmMap[parserType] = item
					}
				}
			}
			if err != nil {
				log.Error("doRefund err: ", parserType, paymentAddress, err.Error())
				notify.SendLarkErrNotify("doRefund", err.Error())
			}
		}
	}
	// stripe
	if err = t.doRefundStripe(stripeList); err != nil {
		log.Error("doRefundStripe err: ", err.Error())
		notify.SendLarkErrNotify("doRefundStripe", err.Error())
	}
	// dp
	if err = t.doRefundDP(dpList); err != nil {
		log.Error("doRefundDP err: %s", err.Error())
		notify.SendLarkErrNotify("doRefundDP", err.Error())
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
	if t.chainEth != nil {
		for k, _ := range config.Cfg.Chain.Eth.AddrMap {
			nonce, err := t.chainEth.NonceAt(k)
			if err != nil {
				return nil, fmt.Errorf("NonceAt eth err: %s", err.Error())
			}
			parserTypeETH.nonceMap[strings.ToLower(k)] = nonce
			nonceInfo, err := t.DbDao.GetRefundNonce(nonce, k, []tables.PayTokenId{tables.PayTokenIdETH, tables.PayTokenIdErc20USDT})
			if err != nil {
				return nil, fmt.Errorf("GetRefundNonce2 eth err: %s[%d][%s]", err.Error(), nonce, k)
			} else if nonceInfo.Id > 0 {
				parserTypeETH.refund = false
				log.Warn("getParserTypeEvmMap eth nonce pending")
			}
		}
	}
	parserTypeEvmMap[tables.ParserTypeETH] = parserTypeETH

	// bsc
	parserTypeBSC := parserTypeEvm{
		addFee:   config.Cfg.Chain.Bsc.RefundAddFee,
		refund:   config.Cfg.Chain.Bsc.Refund,
		chainEvm: t.chainBsc,
		nonceMap: make(map[string]uint64),
	}
	if t.chainBsc != nil {
		for k, _ := range config.Cfg.Chain.Bsc.AddrMap {
			nonce, err := t.chainBsc.NonceAt(k)
			if err != nil {
				return nil, fmt.Errorf("NonceAt bsc err: %s", err.Error())
			}
			parserTypeBSC.nonceMap[strings.ToLower(k)] = nonce
			nonceInfo, err := t.DbDao.GetRefundNonce(nonce, k, []tables.PayTokenId{tables.PayTokenIdBNB, tables.PayTokenIdBep20USDT})
			if err != nil {
				return nil, fmt.Errorf("GetRefundNonce bsc err: %s[%d][%s]", err.Error(), nonce, k)
			} else if nonceInfo.Id > 0 {
				parserTypeBSC.refund = false
				log.Warn("getParserTypeEvmMap bsc nonce pending")
			}
		}
	}
	parserTypeEvmMap[tables.ParserTypeBSC] = parserTypeBSC

	// polygon
	parserTypePolygon := parserTypeEvm{
		addFee:   config.Cfg.Chain.Polygon.RefundAddFee,
		refund:   config.Cfg.Chain.Polygon.Refund,
		chainEvm: t.chainPolygon,
		nonceMap: make(map[string]uint64),
	}
	if t.chainPolygon != nil {
		for k, _ := range config.Cfg.Chain.Polygon.AddrMap {
			nonce, err := t.chainPolygon.NonceAt(k)
			if err != nil {
				return nil, fmt.Errorf("NonceAt polygon err: %s", err.Error())
			}
			parserTypePolygon.nonceMap[strings.ToLower(k)] = nonce
			nonceInfo, err := t.DbDao.GetRefundNonce(nonce, k, []tables.PayTokenId{tables.PayTokenIdMATIC})
			if err != nil {
				return nil, fmt.Errorf("GetRefundNonce polygon err: %s[%d][%s]", err.Error(), nonce, k)
			} else if nonceInfo.Id > 0 {
				parserTypePolygon.refund = false
				log.Warn("getParserTypeEvmMap polygon nonce pending")
			}
		}
	}
	parserTypeEvmMap[tables.ParserTypePOLYGON] = parserTypePolygon

	return parserTypeEvmMap, nil
}
