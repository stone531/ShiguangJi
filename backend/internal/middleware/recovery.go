package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestID 为每个请求生成短随机 id，挂到上下文与响应头。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := newRequestID()
		c.Set("requestID", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func newRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return time.Now().Format("150405.000")
	}
	return hex.EncodeToString(b)
}

// Recover panic 转 500 + 结构化日志（Gin 自带 recovery 不带 requestID，故自实现）。
func Recover() gin.HandlerFunc {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	return func(c *gin.Context) {
		start := time.Now()
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					"requestID", c.GetString("requestID"),
					"path", c.Request.URL.Path,
					"panic", r)
				c.AbortWithStatusJSON(500, gin.H{"code": 50000, "msg": "服务器内部错误"})
			}
			logger.Info("request",
				"requestID", c.GetString("requestID"),
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"status", c.Writer.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
				"ip", c.ClientIP())
		}()
		c.Next()
	}
}
