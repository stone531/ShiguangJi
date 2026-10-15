// SPA 托管与 SEO meta 注入（spec §8.1）。
package router

import (
	"html"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"shiguangji/internal/config"
	"shiguangji/internal/model"
)

// spaStatic 启动时缓存 dist/index.html；NoRoute 按 path 分流。
// dist 不存在（开发态）时全部维持 JSON 404，由 Vite dev server 托管前端。
type spaStatic struct {
	db    *gorm.DB
	index []byte
}

func newSPAStatic(db *gorm.DB, cfg *config.Config) *spaStatic {
	s := &spaStatic{db: db}
	if cfg.WebDist == "" {
		return s
	}
	if data, err := os.ReadFile(filepath.Join(cfg.WebDist, "index.html")); err == nil {
		s.index = data
	}
	return s
}

// handle NoRoute 总入口：/api/* JSON 404；/post/:slug 注入 meta；其余 SPA fallback。
func (s *spaStatic) handle(c *gin.Context) {
	if strings.HasPrefix(c.Request.URL.Path, "/api/") {
		c.JSON(http.StatusNotFound, gin.H{"code": 40004, "msg": "not found"})
		return
	}
	if len(s.index) == 0 { // 开发态无 dist
		c.JSON(http.StatusNotFound, gin.H{"code": 40004, "msg": "not found"})
		return
	}

	if slug, ok := postSlug(c.Request.URL.Path); ok {
		var p model.Post
		err := s.db.WithContext(c.Request.Context()).
			Where("slug = ? AND status = ?", slug, model.StatusPublished).
			First(&p).Error
		if err != nil {
			// 不存在/未发布：仍回 SPA（前端渲染 404 页），但状态码 404 防搜索引擎收录
			c.Data(http.StatusNotFound, "text/html; charset=utf-8", s.index)
			return
		}
		desc := p.Summary
		if desc == "" {
			desc = p.Title
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8",
			injectSEO(s.index, p.Title, desc, p.Cover))
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", s.index)
}

// postSlug 匹配 /post/:slug（slug 内不含 /）。
func postSlug(path string) (string, bool) {
	const prefix = "/post/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	slug := strings.Trim(path[len(prefix):], "/")
	if slug == "" || strings.Contains(slug, "/") {
		return "", false
	}
	return slug, true
}

// injectSEO 替换 <title> 并在 </head> 前插入 description/OG 标签（内容全部转义）。
func injectSEO(tpl []byte, title, desc, image string) []byte {
	out := string(tpl)
	if newTitle := "<title>" + html.EscapeString(title) + " - 时光集</title>"; strings.Contains(out, "<title>") {
		start := strings.Index(out, "<title>")
		end := strings.Index(out[start:], "</title>")
		if end >= 0 {
			out = out[:start] + newTitle + out[start+end+len("</title>"):]
		}
	}

	var b strings.Builder
	b.WriteString(`<meta name="description" content="` + html.EscapeString(desc) + `">`)
	b.WriteString(`<meta property="og:title" content="` + html.EscapeString(title) + `">`)
	b.WriteString(`<meta property="og:description" content="` + html.EscapeString(desc) + `">`)
	b.WriteString(`<meta property="og:type" content="article">`)
	if image != "" {
		b.WriteString(`<meta property="og:image" content="` + html.EscapeString(image) + `">`)
	}
	return []byte(strings.Replace(out, "</head>", b.String()+"</head>", 1))
}
