// 开发模式 /uploads 静态文件服务（生产由 Nginx 直接托管，见 spec §3 架构）。
package router

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"shiguangji/internal/config"
)

// neuteredFileSystem 包装 http.FileSystem，禁用目录列举：目录请求返回 404。
type neuteredFileSystem struct{ fs http.FileSystem }

// Open 实现 http.FileSystem；目录被遮蔽为 os.ErrNotExist。
func (n neuteredFileSystem) Open(path string) (http.File, error) {
	f, err := n.fs.Open(path)
	if err != nil {
		return nil, err
	}
	if st, err := f.Stat(); err == nil && st.IsDir() {
		f.Close()
		return nil, os.ErrNotExist
	}
	return f, nil
}

// registerUploadsStatic 注册 /uploads 静态服务（公开、与 /api 组同级）。
func registerUploadsStatic(r *gin.Engine, cfg *config.Config) {
	r.GET("/uploads/*filepath", gin.WrapH(
		http.StripPrefix("/uploads", http.FileServer(neuteredFileSystem{fs: http.Dir(cfg.UploadDir)})),
	))
}
