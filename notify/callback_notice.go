package notify

import (
	"fmt"
	"github.com/parnurzeal/gorequest"
	"time"
	"unipay/config"
	"unipay/dao"
	"unipay/tables"
)

type CallbackNotice struct {
	DbDao *dao.DbDao
}

func (c *CallbackNotice) CallbackNotice(notice tables.TableNoticeInfo, paymentInfo tables.TablePaymentInfo, orderInfo tables.TableOrderInfo) error {
	// get callback url
	callbackUrl, ok := config.Cfg.BusinessIds[orderInfo.BusinessId]
	if !ok {
		return fmt.Errorf("not exist business id[%s]", orderInfo.BusinessId)
	}

	// send notice
	req := reqCallbackNotice{
		BusinessId: orderInfo.BusinessId,
		EventList: []EventInfo{{
			EventType:    notice.EventType,
			OrderId:      orderInfo.OrderId,
			PayStatus:    orderInfo.PayStatus,
			PayHash:      paymentInfo.PayHash,
			RefundStatus: paymentInfo.RefundStatus,
			RefundHash:   paymentInfo.RefundHash,
		}},
	}
	resp := &respCallbackNotice{}
	if err := doNoticeReq(callbackUrl, req, resp); err != nil {
		return fmt.Errorf("doNoticeReq err: %s", err.Error())
	}
	return nil
}

func (c *CallbackNotice) RepeatCallbackNotice(eventMap map[string][]EventInfo) error {
	if len(eventMap) == 0 {
		return nil
	}

	// callback
	for k, list := range eventMap {
		req := reqCallbackNotice{
			BusinessId: k,
			EventList:  list,
		}
		callbackUrl, ok := config.Cfg.BusinessIds[k]
		if !ok {
			log.Error("BusinessId not exist:", k)
			continue
		}
		resp := &respCallbackNotice{}
		if err := doNoticeReq(callbackUrl, req, resp); err != nil {
			log.Error("doNoticeReq err:", err.Error())
			for _, v := range list {
				noticeCount := v.NoticeCount + 1
				if err := c.DbDao.UpdateNoticeCount(v.NoticeId, noticeCount); err != nil {
					log.Error("UpdateNoticeCount err: ", err.Error(), v.NoticeId)
				}
			}
			continue
		}
		//
		var ids []uint64
		for _, v := range list {
			ids = append(ids, v.NoticeId)
		}
		if err := c.DbDao.UpdateNoticeStatusToOK(ids); err != nil {
			log.Error("UpdateNoticeStatusToOK err:", err.Error(), ids)
		}
	}

	return nil
}

func (c *CallbackNotice) GetEventInfo(notice tables.TableNoticeInfo) (businessId string, eventInfo EventInfo, e error) {
	// check timestamp
	timestamp := notice.Timestamp
	switch notice.NoticeCount {
	case 0: // 30s
		timestamp += 30 * 1e3
	case 1: // 60s
		timestamp += 60 * 1e3
	case 2: // 10 min
		timestamp += 120 * 1e3
	case 3: // 6 h
		timestamp += 300 * 1e3
	case 4:
		// todo 1 d
		timestamp += 300 * 1e3
	default:
		SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "UpdateNoticeStatusToFail", notice.PayHash)
		if err := c.DbDao.UpdateNoticeStatusToFail(notice.Id); err != nil {
			e = fmt.Errorf("UpdateNoticeStatusToFail err: %s", err.Error())
			return
		}
		return
	}
	nowT := time.Now().UnixMilli()
	if nowT < timestamp {
		log.Info("callbackNotice NoticeCount:", notice.NoticeCount, nowT, timestamp-nowT)
		return
	}

	// get payment info
	paymentInfo, err := c.DbDao.GetPaymentInfoByPayHash(notice.PayHash)
	if err != nil {
		e = fmt.Errorf("GetPaymentInfoByPayHash err: %s", err.Error())
		return
	} else if paymentInfo.Id == 0 {
		e = fmt.Errorf("payment not exist[%s]", notice.PayHash)
		return
	}

	// get order info
	orderInfo, err := c.DbDao.GetOrderInfoByOrderId(paymentInfo.OrderId)
	if err != nil {
		e = fmt.Errorf("GetOrderInfoByOrderId err: %s", err.Error())
		return
	} else if orderInfo.Id == 0 {
		e = fmt.Errorf("order not exist[%s]", paymentInfo.OrderId)
		return
	}

	eventInfo = EventInfo{
		EventType:    notice.EventType,
		OrderId:      orderInfo.OrderId,
		PayStatus:    orderInfo.PayStatus,
		PayHash:      paymentInfo.PayHash,
		RefundStatus: paymentInfo.RefundStatus,
		RefundHash:   paymentInfo.RefundHash,
		NoticeId:     notice.Id,
		NoticeCount:  notice.NoticeCount,
	}
	businessId = orderInfo.BusinessId
	return
}

type reqCallbackNotice struct {
	BusinessId string      `json:"business_id"`
	EventList  []EventInfo `json:"event_list"`
}
type EventInfo struct {
	EventType    tables.EventType    `json:"event_type"`
	OrderId      string              `json:"order_id"`
	PayStatus    tables.PayStatus    `json:"pay_status"`
	PayHash      string              `json:"pay_hash"`
	RefundStatus tables.RefundStatus `json:"refund_status"`
	RefundHash   string              `json:"refund_hash"`
	NoticeId     uint64              `json:"notice_id"`
	NoticeCount  int                 `json:"notice_count"`
}
type respCallbackNotice struct {
}

type apiResp struct {
	ErrNo  int         `json:"err_no"`
	ErrMsg string      `json:"err_msg"`
	Data   interface{} `json:"data"`
}

func doNoticeReq(url string, req, data interface{}) error {
	var resp apiResp
	resp.Data = &data

	_, _, errs := gorequest.New().Post(url).
		Timeout(time.Second*10).
		Retry(3, time.Second).
		SendStruct(&req).EndStruct(&resp)
	if len(errs) > 0 {
		return fmt.Errorf("%v", errs)
	}
	if resp.ErrNo != 0 {
		return fmt.Errorf("%d - %s", resp.ErrNo, resp.ErrMsg)
	}
	return nil
}
