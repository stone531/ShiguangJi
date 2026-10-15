package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/glebarez/sqlite"

	"shiguangji/internal/config"
	"shiguangji/internal/model"
)

func newSPATestEnv(t *testing.T, withDist bool) (*gin.Engine, *spaStatic) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(&model.Post{}, &model.Category{}, &model.Tag{}, &model.PostTag{},
		&model.Comment{}, &model.Like{}, &model.SiteSetting{}, &model.Admin{}); err != nil {
		t.Fatal(err)
	}

	dist := ""
	if withDist {
		dist = t.TempDir()
		tpl := "<!doctype html><html><head><meta charset=\"utf-8\"><title>时光集</title></head><body><div id=\"app\"></div></body></html>"
		if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte(tpl), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := &testCfg{dist: dist}
	s := newSPAStatic(db, cfg.toConfig())
	return gin.New(), s
}

type testCfg struct{ dist string }

func (t *testCfg) toConfig() *config.Config { return &config.Config{WebDist: t.dist} }

func TestSPANoRouteRouting(t *testing.T) {
	r, s := newSPATestEnv(t, true)
	r.NoRoute(s.handle)

	db := s.db
	db.Create(&model.Post{Title: "Go 并发笔记", Slug: "go-notes", Content: "x",
		Status: model.StatusPublished, Summary: "并发模型总结", Cover: "/uploads/cover.png"})
	db.Create(&model.Post{Title: "草稿", Slug: "draft", Content: "x", Status: model.StatusDraft})

	cases := []struct {
		path       string
		wantStatus int
		wantIn     []string
		notWant    string
	}{
		{"/api/whatever", 404, []string{`"code":40004`}, ""}, // API JSON 404
		{"/post/go-notes", 200, []string{"<title>Go 并发笔记 - 时光集</title>", // 注入标题
			`<meta name="description" content="并发模型总结">`,
			`<meta property="og:title" content="Go 并发笔记">`,
			`<meta property="og:image" content="/uploads/cover.png">`}, ""},
		{"/post/draft", 404, []string{"<title>时光集</title>"}, "og:title"}, // 未发布：原样 + 404
		{"/post/none", 404, []string{"<title>时光集</title>"}, "og:title"},  // 不存在
		{"/about", 200, []string{"<title>时光集</title>"}, "og:title"},      // SPA fallback 原样
		{"/post/", 200, []string{"<title>时光集</title>"}, "og:title"},      // 空 slug → fallback
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, c.path, nil)
		r.ServeHTTP(w, req)
		if w.Code != c.wantStatus {
			t.Errorf("%s status = %d, want %d", c.path, w.Code, c.wantStatus)
		}
		body := w.Body.String()
		for _, want := range c.wantIn {
			if !strings.Contains(body, want) {
				t.Errorf("%s body missing %q", c.path, want)
			}
		}
		if c.notWant != "" && strings.Contains(body, c.notWant) {
			t.Errorf("%s body should not contain %q", c.path, c.notWant)
		}
	}
}

func TestSPADevWithoutDist(t *testing.T) {
	r, s := newSPATestEnv(t, false)
	r.NoRoute(s.handle)

	for _, path := range []string{"/about", "/post/go-notes"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != 404 || !strings.Contains(w.Body.String(), `"code":40004`) {
			t.Errorf("%s = %d %s, want JSON 404", path, w.Code, w.Body.String())
		}
	}
}

func TestInjectSEOEscapesContent(t *testing.T) {
	tpl := []byte("<html><head><title>x</title></head></html>")
	out := string(injectSEO(tpl, `标题"注入'<script>`, `描述"&<`, ""))
	head := strings.SplitN(out, "</head>", 2)[0]
	if !strings.Contains(head, "&lt;script&gt;") {
		t.Fatalf("script tag not escaped: %s", head)
	}
	if strings.Contains(head, `<meta name="description" content="描述"&<">`) {
		t.Fatalf("raw quote leaked into attribute: %s", head)
	}
	if !strings.Contains(head, `content="描述&#34;&amp;&lt;"`) {
		t.Fatalf("description not properly escaped: %s", head)
	}
}
