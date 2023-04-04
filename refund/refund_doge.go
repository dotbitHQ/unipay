package refund

import (
	"unipay/config"
	"unipay/tables"
)

func (t *ToolRefund) doRefundDoge(list []tables.TablePaymentInfo) error {
	if !config.Cfg.Chain.Ckb.Refund {
		return nil
	}
	return nil
	//var orderIds []string
	//for _, v := range list {
	//	orderIds = append(orderIds, v.OrderId)
	//}
	//// check order
	//orders, err := d.DbDao.GetOrders(orderIds)
	//if err != nil {
	//	return "", fmt.Errorf("GetOrders err: %s", err.Error())
	//}
	//var notRefundMap = make(map[string]struct{})
	//for _, v := range orders {
	//	if v.Action == common.DasActionApplyRegister && v.RegisterStatus == tables.RegisterStatusRegistered {
	//		notRefundMap[v.OrderId] = struct{}{}
	//	}
	//}
	////
	//var hashList []string
	//var addresses []string
	//var values []int64
	//var total int64
	//for _, v := range list {
	//	if _, ok := notRefundMap[v.OrderId]; ok {
	//		log.Warn("notRefundMap:", v.OrderId)
	//		notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doOrderRefundDoge", "notRefundMap: "+v.OrderId)
	//		continue
	//	}
	//	hashList = append(hashList, v.Hash)
	//	addr, err := common.Base58CheckEncode(v.Address, common.DogeCoinBase58Version)
	//	if err != nil {
	//		return "", fmt.Errorf("Base58CheckEncode err: %s", err.Error())
	//	}
	//	addresses = append(addresses, addr)
	//	value := v.PayAmount.IntPart()
	//	total += value
	//	values = append(values, value)
	//}
	//if len(addresses) == 0 || len(values) == 0 {
	//	return "", nil
	//}
	//
	//// get utxo
	//_, uos, err := d.ChainDoge.GetUnspentOutputsDoge(config.Cfg.Chain.Doge.Address, config.Cfg.Chain.Doge.Private, total)
	//if err != nil {
	//	return "", fmt.Errorf("GetUnspentOutputsDoge err: %s", err.Error())
	//}
	//
	//// build tx
	//tx, err := d.ChainDoge.NewTx(uos, addresses, values, "")
	//if err != nil {
	//	return "", fmt.Errorf("NewTx err: %s", err.Error())
	//}
	//
	//// sign
	//var signTx *wire.MsgTx
	//if config.Cfg.Chain.Doge.Private != "" {
	//	_, err = d.ChainDoge.LocalSignTx(tx, uos)
	//	if err != nil {
	//		return "", fmt.Errorf("LocalSignTx err: %s", err.Error())
	//	}
	//	signTx = tx
	//} else if d.ChainDoge.RemoteSignClient != nil {
	//	signTx, err = d.ChainDoge.RemoteSignTx(bitcoin.RemoteSignMethodDogeTx, tx, uos)
	//	if err != nil {
	//		return "", fmt.Errorf("LocalSignTx err: %s", err.Error())
	//	}
	//} else {
	//	return "", fmt.Errorf("no signature configured")
	//}
	//
	////
	//if err := d.DbDao.UpdateRefundStatus(hashList, tables.TxStatusSending, tables.TxStatusOk); err != nil {
	//	return "", fmt.Errorf("UpdateRefundStatus err: %s", err.Error())
	//}
	//// send tx
	//hash, err := d.ChainDoge.SendTx(signTx)
	//if err != nil {
	//	if err := d.DbDao.UpdateRefundStatus(hashList, tables.TxStatusOk, tables.TxStatusSending); err != nil {
	//		log.Info("UpdateRefundStatus err: ", err.Error(), hashList)
	//		notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "order refund doge", notify.GetLarkTextNotifyStr("UpdateRefundStatus", strings.Join(hashList, ","), err.Error()))
	//	}
	//	return "", fmt.Errorf("SendTx err: %s", err.Error())
	//}
	//if err := d.DbDao.UpdateRefundHash(hashList, hash); err != nil {
	//	log.Info("UpdateRefundHash err:", err.Error(), hashList, hash)
	//	notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "order refund doge", notify.GetLarkTextNotifyStr("UpdateRefundHash", strings.Join(hashList, ",")+";"+hash, err.Error()))
	//}
	//
	//return hash, nil

}
