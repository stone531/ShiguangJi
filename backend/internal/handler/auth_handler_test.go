package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"shiguangji/internal/config"
	"shiguangji/internal/model"
	"shiguangji/internal/router"
)

func setupAuth(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Admin{}); err != nil {
		t.Fatal(err)
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("right-pass"), bcrypt.MinCost)
	db.Create(&model.Admin{Username: "admin", PasswordHash: string(hash), Nickname: "博主"})

	cfg := &config.Config{JWTSecret: "test-secret", JWTTimeout: time.Hour}
	r := gin.New()
	router.RegisterAuthRoutes(r, db, cfg) // 测试装配入口（不挂 RequestID/Recover）
	return r, db
}

func TestLoginSuccess(t *testing.T) {
	r, _ := setupAuth(t)
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "right-pass"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("code = %d, body = %s", w.Code, w.Body.String())
	}
	var resp struct{ Code int }
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 0 {
		t.Fatalf("biz code = %d", resp.Code)
	}
	// HttpOnly cookie 必须下发
	found := false
	for _, ck := range w.Result().Cookies() {
		if ck.Name == "blog_token" && ck.HttpOnly {
			found = true
		}
	}
	if !found {
		t.Fatal("blog_token HttpOnly cookie not set")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	r, _ := setupAuth(t)
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "wrong-password"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	var resp struct{ Code int }
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code == 0 {
		t.Fatal("wrong password must fail")
	}
}

func TestAdminRouteRejectsNoCookie(t *testing.T) {
	r, _ := setupAuth(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", w.Code)
	}
}
