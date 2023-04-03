package notify

import (
	"fmt"
	"github.com/dotbitHQ/unipay/config"
	"github.com/dotbitHQ/unipay/dao"
	"github.com/dotbitHQ/unipay/http_svr/api_code"
	"github.com/dotbitHQ/unipay/tables"
	"github.com/parnurzeal/gorequest"
	"time"
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
		OrderId:   notice.OrderId,
		EventType: notice.EventType,
		OrderInfo: reqOrderInfo{
			PayStatus:    orderInfo.PayStatus,
			PayHash:      paymentInfo.PayHash,
			RefundStatus: paymentInfo.RefundStatus,
			RefundHash:   paymentInfo.RefundHash,
		},
	}
	resp := &respCallbackNotice{}
	if err := doNoticeReq(callbackUrl, req, resp); err != nil {
		return fmt.Errorf("doNoticeReq err: %s", err.Error())
	}
	return nil
}

func (c *CallbackNotice) RepeatCallbackNotice(notice tables.TableNoticeInfo) error {
	// check timestamp
	timestamp := notice.Timestamp
	switch notice.NoticeCount {
	case 0: // 30s
		timestamp += 30
	case 1: // 60s
		timestamp += 60
	case 2: // 120s
		timestamp += 120
	case 3: // 300s
		timestamp += 300
	default:
		//if err := c.DbDao.UpdateNoticeStatus(notice.Id, tables.NoticeStatusDefault, tables.NoticeStatusFail); err != nil {
		//	return fmt.Errorf("UpdateNoticeStatus err: %s", err.Error())
		//}
		return nil
	}
	nowT := time.Now().Unix()
	if nowT < timestamp {
		log.Info("callbackNotice NoticeCount:", notice.NoticeCount, timestamp-nowT)
		return nil
	}

	// get order info
	orderInfo, err := c.DbDao.GetOrderInfoByOrderId(notice.OrderId)
	if err != nil {
		return fmt.Errorf("GetOrderInfoByOrderId err: %s", err.Error())
	} else if orderInfo.Id == 0 {
		return fmt.Errorf("order not exist[%s]", notice.OrderId)
	}

	// get payment info
	paymentInfo, err := c.DbDao.GetLatestPaymentInfo(notice.OrderId)
	if err != nil {
		return fmt.Errorf("GetLatestPaymentInfo err: %s", err.Error())
	} else if paymentInfo.Id == 0 {
		return fmt.Errorf("payment not exist[%s]", notice.OrderId)
	}

	// callback
	if err := c.CallbackNotice(notice, paymentInfo, orderInfo); err != nil {
		notice.NoticeCount++
		if notice.NoticeCount > 3 {
			// notify
			txt := fmt.Sprintf(`BusinessId: %s
OrderId: %s`, orderInfo.BusinessId, notice.OrderId)
			SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "RepeatCallbackNotice", txt)
			if err := c.DbDao.UpdateNoticeStatus(notice.Id, tables.NoticeStatusDefault, tables.NoticeStatusFail); err != nil {
				return fmt.Errorf("UpdateNoticeStatus err: %s", err.Error())
			}
		} else {
			if err := c.DbDao.UpdateNoticeCount(notice); err != nil {
				return fmt.Errorf("UpdateNoticeCount err: %s", err.Error())
			}
		}
		return fmt.Errorf("CallbackNotice err: %s", err.Error())
	}

	// update status, retry
	if err := c.DbDao.UpdateNoticeStatus(notice.Id, tables.NoticeStatusDefault, tables.NoticeStatusOK); err != nil {
		return fmt.Errorf("UpdateNoticeStatus err: %s", err.Error())
	}

	return nil
}

type reqCallbackNotice struct {
	OrderId   string           `json:"order_id"`
	EventType tables.EventType `json:"event_type"`
	OrderInfo reqOrderInfo     `json:"order_info"`
}
type reqOrderInfo struct {
	PayStatus    tables.PayStatus    `json:"pay_status"`
	PayHash      string              `json:"pay_hash"`
	RefundStatus tables.RefundStatus `json:"refund_status"`
	RefundHash   string              `json:"refund_hash"`
}
type respCallbackNotice struct {
}

func doNoticeReq(url string, req, data interface{}) error {
	var resp api_code.ApiResp
	resp.Data = &data

	_, _, errs := gorequest.New().Post(url).
		Timeout(time.Second*10).
		Retry(3, time.Second).
		SendStruct(&req).EndStruct(&resp)
	if len(errs) > 0 {
		return fmt.Errorf("%v", errs)
	}
	if resp.ErrNo != api_code.ApiCodeSuccess {
		return fmt.Errorf("%d - %s", resp.ErrNo, resp.ErrMsg)
	}
	return nil
}
