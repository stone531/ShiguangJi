package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestLimiterAllowsWithinLimit(t *testing.T) {
	l := NewIPLimiter(5, time.Minute)
	defer l.Close()
	for i := 0; i < 5; i++ {
		if !l.Allow("1.2.3.4") {
			t.Fatalf("request %d should pass", i+1)
		}
	}
	if l.Allow("1.2.3.4") {
		t.Fatal("6th request should be blocked")
	}
	if !l.Allow("5.6.7.8") {
		t.Fatal("different IP should not be affected")
	}
}

func TestLimiterCleansExpired(t *testing.T) {
	l := NewIPLimiter(1, 10*time.Millisecond)
	defer l.Close()
	l.Allow("1.1.1.1")
	time.Sleep(20 * time.Millisecond)
	l.cleanup()
	if !l.Allow("1.1.1.1") {
		t.Fatal("entry should be cleaned after expiry")
	}
}

func TestLimiterWindowReset(t *testing.T) {
	l := NewIPLimiter(2, 50*time.Millisecond)
	defer l.Close()
	for i := 0; i < 2; i++ {
		if !l.Allow("1.1.1.1") {
			t.Fatalf("request %d should pass", i+1)
		}
	}
	if l.Allow("1.1.1.1") {
		t.Fatal("3rd request in window should be blocked")
	}
	time.Sleep(60 * time.Millisecond)
	if !l.Allow("1.1.1.1") {
		t.Fatal("new window should reset count")
	}
}

func TestLimiterGinMiddleware429(t *testing.T) {
	gin.SetMode(gin.TestMode)
	l := NewIPLimiter(1, time.Minute)
	defer l.Close()
	r := gin.New()
	r.POST("/login", l.GinMiddleware(), func(c *gin.Context) { c.JSON(200, gin.H{}) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", nil)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("first request code = %d, want 200", w.Code)
	}

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest("POST", "/login", nil))
	if w2.Code != 429 {
		t.Fatalf("second request code = %d, want 429", w2.Code)
	}
	if !strings.Contains(w2.Body.String(), "42901") {
		t.Fatalf("body should contain rate-limit code, got %s", w2.Body.String())
	}
}
