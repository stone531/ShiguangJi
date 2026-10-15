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
	"gorm.io/gorm"

	"shiguangji/internal/config"
	"shiguangji/internal/middleware"
	"shiguangji/internal/model"
	"shiguangji/internal/pkg/jwtutil"
	"shiguangji/internal/router"
)

// setupInteraction 评论/点赞 HTTP 层测试装配：发布文章 + 管理员（id=1）。
// httptest 默认 RemoteAddr 为同一 IP，正好用于触发同 IP 限流。
func setupInteraction(t *testing.T) (*gin.Engine, *gorm.DB, *config.Config) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Admin{}, &model.Post{}, &model.Comment{},
		&model.Like{}, &model.SiteSetting{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&model.Admin{Username: "admin", PasswordHash: "x", Nickname: "博主"})
	db.Create(&model.Post{Title: "p", Slug: "p", Content: "x", Status: model.StatusPublished})

	cfg := &config.Config{JWTSecret: "test-secret", JWTTimeout: time.Hour, LikeSalt: "salt"}
	r := gin.New()
	router.RegisterAuthRoutes(r, db, cfg)
	return r, db, cfg
}

// adminCookie 直接铸 token（不走登录，避免占用共享登录限流器配额）。
func adminCookie(t *testing.T, cfg *config.Config) *http.Cookie {
	t.Helper()
	tok, err := jwtutil.Issue(cfg.JWTSecret, 1, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: middleware.CookieName, Value: tok.Token}
}

func doJSON(r *gin.Engine, method, url string, body any, xhr bool, cookie *http.Cookie) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, url, reader)
	req.Header.Set("Content-Type", "application/json")
	if xhr {
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder, out any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
		t.Fatalf("bad json %q: %v", w.Body.String(), err)
	}
}

// TestCommentCreateRequiresXHR 公开评论 POST 缺 XHR 头 → 400/40001，且不落库。
func TestCommentCreateRequiresXHR(t *testing.T) {
	r, db, _ := setupInteraction(t)
	w := doJSON(r, http.MethodPost, "/api/posts/p/comments",
		map[string]string{"nickname": "访客", "content": "你好"}, false, nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", w.Code, w.Body.String())
	}
	var resp struct{ Code int }
	decode(t, w, &resp)
	if resp.Code != 40001 {
		t.Fatalf("biz code = %d, want 40001", resp.Code)
	}
	var n int64
	db.Model(&model.Comment{}).Count(&n)
	if n != 0 {
		t.Fatalf("comments = %d, want 0（被拒请求不得落库）", n)
	}
}

// TestCommentHoneypotFakeSuccess 蜜罐字段非空 → 假成功（200/code 0），不落库。
func TestCommentHoneypotFakeSuccess(t *testing.T) {
	r, db, _ := setupInteraction(t)
	w := doJSON(r, http.MethodPost, "/api/posts/p/comments",
		map[string]string{"nickname": "bot", "content": "spam", "website": "http://spam.example"}, true, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200（蜜罐需伪装成功）", w.Code)
	}
	var resp struct{ Code int }
	decode(t, w, &resp)
	if resp.Code != 0 {
		t.Fatalf("biz code = %d, want 0", resp.Code)
	}
	var n int64
	db.Model(&model.Comment{}).Count(&n)
	if n != 0 {
		t.Fatalf("comments = %d, want 0（蜜罐命中静默丢弃）", n)
	}
}

// TestCommentCreateAndRateLimit 合法评论 200 且待审落库；同 IP 同文章第二条 → HTTP 429/42901。
func TestCommentCreateAndRateLimit(t *testing.T) {
	r, db, _ := setupInteraction(t)

	w := doJSON(r, http.MethodPost, "/api/posts/p/comments",
		map[string]string{"nickname": "访客", "content": "写得不错"}, true, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("first comment status = %d, body = %s", w.Code, w.Body.String())
	}
	var cm model.Comment
	if err := db.First(&cm).Error; err != nil {
		t.Fatal("comment not persisted")
	}
	if cm.Status != model.CommentPending {
		t.Fatalf("status = %d, want pending（先审后显）", cm.Status)
	}

	w = doJSON(r, http.MethodPost, "/api/posts/p/comments",
		map[string]string{"nickname": "访客", "content": "再来一条"}, true, nil)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("second comment status = %d, want 429", w.Code)
	}
	var resp struct{ Code int }
	decode(t, w, &resp)
	if resp.Code != 42901 {
		t.Fatalf("biz code = %d, want 42901", resp.Code)
	}
}

// TestLikeRequiresXHRAndRoundTrip 点赞 POST 缺头 400；带头切换往返 + GET 状态。
func TestLikeRequiresXHRAndRoundTrip(t *testing.T) {
	r, _, _ := setupInteraction(t)

	if w := doJSON(r, http.MethodPost, "/api/posts/p/like", nil, false, nil); w.Code != http.StatusBadRequest {
		t.Fatalf("no-xhr like status = %d, want 400", w.Code)
	}

	var like struct {
		Code int `json:"code"`
		Data struct {
			Liked     bool  `json:"liked"`
			LikeCount int64 `json:"like_count"`
		} `json:"data"`
	}
	w := doJSON(r, http.MethodPost, "/api/posts/p/like", nil, true, nil)
	decode(t, w, &like)
	if like.Code != 0 || !like.Data.Liked || like.Data.LikeCount != 1 {
		t.Fatalf("toggle-on = %+v", like)
	}
	w = doJSON(r, http.MethodPost, "/api/posts/p/like", nil, true, nil)
	decode(t, w, &like)
	if like.Code != 0 || like.Data.Liked || like.Data.LikeCount != 0 {
		t.Fatalf("toggle-off = %+v", like)
	}
	w = doJSON(r, http.MethodGet, "/api/posts/p/like", nil, false, nil)
	decode(t, w, &like)
	if like.Code != 0 || like.Data.Liked || like.Data.LikeCount != 0 {
		t.Fatalf("status = %+v", like)
	}
}

// TestAdminCommentsAuthAndReview 后台评论路由：无 cookie 401；铸 token 后列表可见、审核生效、公开列表同步。
func TestAdminCommentsAuthAndReview(t *testing.T) {
	r, db, cfg := setupInteraction(t)
	db.Create(&model.Comment{PostID: 1, Nickname: "访客", Content: "求更新", Status: model.CommentPending})

	// 无认证 → 401/20001
	w := doJSON(r, http.MethodGet, "/api/admin/comments", nil, false, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no-cookie status = %d, want 401", w.Code)
	}

	ck := adminCookie(t, cfg)

	// 列表含待审评论与所属文章标题（联表 VO）
	w = doJSON(r, http.MethodGet, "/api/admin/comments", nil, false, ck)
	if w.Code != http.StatusOK {
		t.Fatalf("admin list status = %d", w.Code)
	}
	var list struct {
		Data struct {
			List []struct {
				ID        int    `json:"id"`
				PostTitle string `json:"post_title"`
				Status    int8   `json:"status"`
			} `json:"list"`
			Total int64 `json:"total"`
		} `json:"data"`
	}
	decode(t, w, &list)
	if list.Data.Total != 1 || len(list.Data.List) != 1 ||
		list.Data.List[0].PostTitle != "p" || list.Data.List[0].Status != model.CommentPending {
		t.Fatalf("admin list = %+v", list.Data)
	}

	// 审核 PUT 需 XHR：带头通过 → 公开评论树立即可见
	if w := doJSON(r, http.MethodPut, "/api/admin/comments/1/review",
		map[string]int{"status": 1}, false, ck); w.Code != http.StatusBadRequest {
		t.Fatalf("review without xhr status = %d, want 400", w.Code)
	}
	w = doJSON(r, http.MethodPut, "/api/admin/comments/1/review",
		map[string]int{"status": 1}, true, ck)
	if w.Code != http.StatusOK {
		t.Fatalf("review status = %d, body = %s", w.Code, w.Body.String())
	}
	w = doJSON(r, http.MethodGet, "/api/posts/p/comments", nil, false, nil)
	var pub struct {
		Data struct {
			List []struct {
				Nickname string `json:"nickname"`
			} `json:"list"`
			Total int64 `json:"total"`
		} `json:"data"`
	}
	decode(t, w, &pub)
	if pub.Data.Total != 1 || len(pub.Data.List) != 1 || pub.Data.List[0].Nickname != "访客" {
		t.Fatalf("public list after review = %+v", pub.Data)
	}
}
