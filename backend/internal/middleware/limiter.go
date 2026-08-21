package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"shiguangji/internal/errs"
)

type ipEntry struct {
	count       int
	windowStart time.Time
	lastSeen    time.Time
}

// IPLimiter 固定窗口限流；cleanup 用 Ticker 而非循环 time.After，防 timer 泄漏。
type IPLimiter struct {
	mu      sync.Mutex
	entries map[string]*ipEntry
	limit   int
	window  time.Duration
	stop    chan struct{}
}

// NewIPLimiter 创建限流器并启动后台清理 goroutine；用完需 Close。
func NewIPLimiter(limit int, window time.Duration) *IPLimiter {
	l := &IPLimiter{
		entries: make(map[string]*ipEntry),
		limit:   limit,
		window:  window,
		stop:    make(chan struct{}),
	}
	go l.cleanupLoop()
	return l
}

// Allow 判断该 IP 本次请求是否放行。
func (l *IPLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	e, ok := l.entries[ip]
	if !ok || now.Sub(e.windowStart) > l.window {
		l.entries[ip] = &ipEntry{count: 1, windowStart: now, lastSeen: now}
		return true
	}
	e.count++
	e.lastSeen = now
	return e.count <= l.limit
}

func (l *IPLimiter) cleanupLoop() {
	ticker := time.NewTicker(l.window)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			l.cleanup()
		case <-l.stop:
			return
		}
	}
}

func (l *IPLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	for ip, e := range l.entries {
		if now.Sub(e.lastSeen) > l.window {
			delete(l.entries, ip)
		}
	}
}

// Close 停止清理 goroutine。
func (l *IPLimiter) Close() { close(l.stop) }

// GinMiddleware 挂在登录路由上。
func (l *IPLimiter) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow(c.ClientIP()) {
			c.AbortWithStatusJSON(429, gin.H{"code": errs.ErrRateLimited.Code, "msg": errs.ErrRateLimited.Msg})
			return
		}
		c.Next()
	}
}
