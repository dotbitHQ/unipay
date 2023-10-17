package notify

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/http_api/logger"
	"github.com/parnurzeal/gorequest"
	"time"
)

var log = logger.NewLogger("notify", logger.LevelDebug)

const (
	LarkNotifyUrl = "https://open.larksuite.com/open-apis/bot/v2/hook/%s"
)

type MsgContent struct {
	Tag      string `json:"tag"`
	UnEscape bool   `json:"un_escape"`
	Text     string `json:"text"`
	UserName string `json:"user_name"`
	UserId   string `json:"user_id"`
}
type MsgData struct {
	Email   string `json:"email"`
	MsgType string `json:"msg_type"`
	Content struct {
		Post struct {
			ZhCn struct {
				Title   string         `json:"title"`
				Content [][]MsgContent `json:"content"`
			} `json:"zh_cn"`
		} `json:"post"`
	} `json:"content"`
}

func SendLarkTextNotify(key, title, text string) {
	if key == "" || text == "" {
		return
	}
	var data MsgData
	data.Email = ""
	data.MsgType = "post"
	data.Content.Post.ZhCn.Title = fmt.Sprintf("UNIPAY: %s", title)
	data.Content.Post.ZhCn.Content = [][]MsgContent{
		{
			MsgContent{
				Tag:      "text",
				UnEscape: false,
				Text:     text,
			},
		},
	}
	url := fmt.Sprintf(LarkNotifyUrl, key)
	_, body, errs := gorequest.New().Post(url).Timeout(time.Second*10).Retry(5, 5*time.Second).SendStruct(&data).End()
	if len(errs) > 0 {
		log.Error("SendLarkTextNotify req err:", errs)
	} else {
		log.Info("SendLarkTextNotify req:", body)
	}
}

func SendLarkTextNotifyAtAll(key, title, text string) {
	if key == "" || text == "" {
		return
	}
	var data MsgData
	data.Email = ""
	data.MsgType = "post"
	data.Content.Post.ZhCn.Title = fmt.Sprintf("UNIPAY: %s", title)
	data.Content.Post.ZhCn.Content = [][]MsgContent{
		{
			MsgContent{
				Tag:      "text",
				UnEscape: false,
				Text:     text,
			},
		},
		{
			MsgContent{
				Tag:      "at",
				UserId:   "all",
				UserName: "所有人",
			},
		},
	}
	url := fmt.Sprintf(LarkNotifyUrl, key)
	_, body, errs := gorequest.New().Post(url).Timeout(time.Second*10).Retry(5, 5*time.Second).SendStruct(&data).End()
	if len(errs) > 0 {
		log.Error("SendLarkTextNotifyAtAll req err:", errs)
	} else {
		log.Info("SendLarkTextNotifyAtAll req:", body)
	}
}

type StripeInfo struct {
	PID         string
	Account     string
	AlgorithmId string
	Address     string
	Action      string
	Amount      int64
}

func SendStripeNotify(key string, si StripeInfo) {
	msg := fmt.Sprintf(`> PID: %s
> Account: %s
> AlgorithmId: %s
> Address: %s
> Action: %s
> Amount: %.2f`, si.PID, si.Account, si.AlgorithmId, si.Address, si.Action, float64(si.Amount)/100)
	go func() {
		SendLarkTextNotify(key, "Stripe Payment", msg)
	}()
}
