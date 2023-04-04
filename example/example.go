package example

import (
	"fmt"
	"github.com/parnurzeal/gorequest"
	"unipay/http_svr/api_code"
)

var ApiUrl = "http://127.0.0.1:9090/v1"

func doReq(url string, req, data interface{}) error {
	var resp api_code.ApiResp
	resp.Data = &data

	_, _, errs := gorequest.New().Post(url).SendStruct(&req).EndStruct(&resp)
	if errs != nil {
		return fmt.Errorf("%v", errs)
	}
	if resp.ErrNo != api_code.ApiCodeSuccess {
		return fmt.Errorf("%d - %s", resp.ErrNo, resp.ErrMsg)
	}
	return nil
}
