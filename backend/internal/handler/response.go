package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"shiguangji/internal/errs"
)

// Response 统一响应结构。
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// OK 成功响应。
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: 0, Msg: "ok", Data: data})
}

// Fail 失败响应：BizError 按业务码返回；限流 42901 特殊走 429，其余业务码 400，未知错误 500。
func Fail(c *gin.Context, err error) {
	var be *errs.BizError
	if errors.As(err, &be) {
		status := http.StatusBadRequest
		if be.Code == errs.ErrRateLimited.Code {
			status = http.StatusTooManyRequests
		}
		c.JSON(status, Response{Code: be.Code, Msg: be.Msg})
		return
	}
	c.JSON(http.StatusInternalServerError, Response{Code: 50000, Msg: "服务器内部错误"})
}
