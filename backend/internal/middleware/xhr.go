package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"shiguangji/internal/errs"
)

// RequireXHR CSRF 纵深防御：非 GET/HEAD/OPTIONS 写请求必须带
// X-Requested-With: XMLHttpRequest。跨站表单/简单请求无法自定义头，
// 配合 Cookie SameSite=Strict 双保险（spec §8）。
func RequireXHR() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		if c.GetHeader("X-Requested-With") != "XMLHttpRequest" {
			c.AbortWithStatusJSON(http.StatusBadRequest,
				gin.H{"code": errs.ErrParam.Code, "msg": "缺少 X-Requested-With 头"})
			return
		}
		c.Next()
	}
}
