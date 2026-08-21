package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"shiguangji/internal/config"
	"shiguangji/internal/pkg/jwtutil"
)

func newTestCfg() *config.Config {
	return &config.Config{JWTSecret: "test-secret", JWTTimeout: time.Hour}
}

func newJWTRouter(cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", JWT(cfg), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"adminID": c.GetUint("adminID")})
	})
	return r
}

func doReq(r *gin.Engine, token string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	if token != "" {
		req.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	}
	r.ServeHTTP(w, req)
	return w
}

func TestJWTNoCookie(t *testing.T) {
	w := doReq(newJWTRouter(newTestCfg()), "")
	if w.Code != 401 {
		t.Fatalf("code = %d, want 401", w.Code)
	}
	if !strings.Contains(w.Body.String(), "20001") {
		t.Fatalf("body should contain 20001, got %s", w.Body.String())
	}
}

func TestJWTValidToken(t *testing.T) {
	cfg := newTestCfg()
	issued, err := jwtutil.Issue(cfg.JWTSecret, 7, cfg.JWTTimeout)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	w := doReq(newJWTRouter(cfg), issued.Token)
	if w.Code != 200 {
		t.Fatalf("code = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"adminID":7`) {
		t.Fatalf("adminID not set, body = %s", w.Body.String())
	}
}

func TestJWTExpiredToken(t *testing.T) {
	cfg := newTestCfg()
	issued, _ := jwtutil.Issue(cfg.JWTSecret, 1, -time.Hour)
	w := doReq(newJWTRouter(cfg), issued.Token)
	if w.Code != 401 {
		t.Fatalf("code = %d, want 401", w.Code)
	}
	if !strings.Contains(w.Body.String(), "20002") {
		t.Fatalf("body should contain 20002, got %s", w.Body.String())
	}
}

func TestJWTRenewsWhenUnderHalfLife(t *testing.T) {
	cfg := newTestCfg()
	// 剩余 10 分钟 < JWTTimeout/2 (30 分钟)，触发续签
	issued, _ := jwtutil.Issue(cfg.JWTSecret, 1, 10*time.Minute)
	w := doReq(newJWTRouter(cfg), issued.Token)
	if w.Code != 200 {
		t.Fatalf("code = %d, want 200", w.Code)
	}
	respCookie := ""
	for _, c := range w.Result().Cookies() {
		if c.Name == CookieName {
			respCookie = c.Value
		}
	}
	if respCookie == "" {
		t.Fatal("response should set new auth cookie")
	}
	if respCookie == issued.Token {
		t.Fatal("reissued token should differ from request token")
	}
}

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/", func(c *gin.Context) { c.JSON(200, gin.H{}) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	h := w.Header()
	checks := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	for k, want := range checks {
		if got := h.Get(k); got != want {
			t.Fatalf("header %s = %q, want %q", k, got, want)
		}
	}
}
