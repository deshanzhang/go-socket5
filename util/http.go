package util

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
)

const (
	SuccessCode = iota + 2000
)

const (
	ErrorCode = iota + 4000
)

type RespInput struct {
	Code      int         `json:"code"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data"`
	Message   string      `json:"message"`
}

func ReturnJson(ctx *gin.Context, resp *RespInput, err error) {
	if resp == nil {
		resp = &RespInput{}
	}
	ctx.Header("Content-Type", "application/json")
	if err != nil {
		resp.Code = ErrorCode
		resp.Data = nil
		resp.Message = err.Error()
	} else {
		resp.Code = SuccessCode
		if resp.Message == "" {
			resp.Message = "请求成功"
		}
	}
	resp.Timestamp = time.Now().Unix()
	jsonStr, _ := json.Marshal(resp)
	ctx.Writer.WriteString(string(jsonStr))
}

func ReResponse(res *resty.Response, resp interface{}) (*RespInput, error) {
	resData := &RespInput{}
	if err := json.Unmarshal(res.Body(), resData); err != nil {
		return resData, err
	}
	ConsoleJson(resData)
	if resData.Code == ErrorCode {
		return resData, nil
	}
	jsonStr := ToJson(resData.Data)
	ConsoleJson(jsonStr)
	return resData, ReJson(jsonStr, resp)
}
