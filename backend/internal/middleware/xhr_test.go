package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestRequireXHR 写方法缺头 400 / 带头放行 / 安全方法（GET/HEAD/OPTIONS）免头放行。
func TestRequireXHR(t *testing.T) {
	gin.SetMode(gin.TestMode)

	do := func(method, xhrHeader string) (int, int) {
		reached := false
		r := gin.New()
		r.Any("/ping", RequireXHR(), func(c *gin.Context) {
			reached = true
			c.Status(http.StatusOK)
		})
		req := httptest.NewRequest(method, "/ping", nil)
		if xhrHeader != "" {
			req.Header.Set("X-Requested-With", xhrHeader)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		code := 0
		if w.Code != http.StatusOK {
			json.Unmarshal(w.Body.Bytes(), &struct{ Code *int }{Code: &code})
		}
		if w.Code == http.StatusOK && !reached {
			t.Fatalf("%s: 200 但业务 handler 未执行", method)
		}
		return w.Code, code
	}

	cases := []struct {
		method, header string
		wantStatus     int
		wantCode       int // 业务码（40001），放行时为 0
	}{
		{http.MethodPost, "", http.StatusBadRequest, 40001},
		{http.MethodPut, "", http.StatusBadRequest, 40001},
		{http.MethodDelete, "", http.StatusBadRequest, 40001},
		{http.MethodPost, "XMLHttpRequest", http.StatusOK, 0},
		{http.MethodPost, "wrong-value", http.StatusBadRequest, 40001},
		{http.MethodGet, "", http.StatusOK, 0},
		{http.MethodHead, "", http.StatusOK, 0},
		{http.MethodOptions, "", http.StatusOK, 0},
	}
	for _, c := range cases {
		status, code := do(c.method, c.header)
		if status != c.wantStatus || code != c.wantCode {
			t.Errorf("%s (header=%q): status=%d code=%d, want %d/%d",
				c.method, c.header, status, code, c.wantStatus, c.wantCode)
		}
	}
}
