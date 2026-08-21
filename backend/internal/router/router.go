// Package router 路由注册。
package router

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"shiguangji/internal/config"
	h "shiguangji/internal/handler"
	"shiguangji/internal/middleware"
	"shiguangji/internal/service"
)

var (
	loginLimiter     *middleware.IPLimiter
	loginLimiterOnce sync.Once
)

// getLoginLimiter 登录限流器只初始化一次（测试与 main 共用同一实例）。
func getLoginLimiter() *middleware.IPLimiter {
	loginLimiterOnce.Do(func() {
		loginLimiter = middleware.NewIPLimiter(5, time.Minute)
	})
	return loginLimiter
}

// CloseAuthLimiter 优雅关停时释放限流器后台 goroutine（main 调用）。
func CloseAuthLimiter() {
	if loginLimiter != nil {
		loginLimiter.Close()
	}
}

// RegisterAuthRoutes 认证路由装配（测试入口，不挂 RequestID/Recover；main 经 RegisterAll 也复用它）。
func RegisterAuthRoutes(r *gin.Engine, db *gorm.DB, cfg *config.Config) {
	authSvc := service.NewAuthService(db, cfg.JWTSecret, cfg.JWTTimeout)
	authH := h.NewAuthHandler(authSvc, cfg)
	limiter := getLoginLimiter()

	// 开发模式静态文件服务；生产由 Nginx 直接托管 /uploads（spec §3 架构）
	registerUploadsStatic(r, cfg)

	api := r.Group("/api")
	api.Use(middleware.SecurityHeaders())
	{
		admin := api.Group("/admin")
		admin.POST("/login", limiter.GinMiddleware(), authH.Login)

		protected := admin.Group("", middleware.JWT(cfg))
		protected.POST("/password", authH.ChangePassword)
		protected.GET("/stats", func(c *gin.Context) { h.OK(c, gin.H{"todo": "M2 仪表盘"}) })

		// 文章管理（admin 路径下 :id，与前台 :slug 不同组，无 Gin 路由冲突）
		postSvc := service.NewPostService(db)
		postH := h.NewPostHandler(postSvc)
		protected.GET("/posts", postH.ListAdmin)
		protected.POST("/posts", postH.Create)
		protected.GET("/posts/:id", postH.GetByID)
		protected.PUT("/posts/:id", postH.Update)
		protected.PUT("/posts/:id/status", postH.SetStatus)
		protected.DELETE("/posts/:id", postH.Delete)

		// 前台公开（/api/posts/:slug）
		api.GET("/posts", postH.ListPublic)
		api.GET("/posts/:slug", postH.GetBySlugPublic)

		// 图片上传（含同步缩略图）
		uploadH := h.NewUploadHandler(service.NewUploadService(cfg.UploadDir, 5<<20))
		protected.POST("/upload", uploadH.Upload)
	}
}

// RegisterAll 完整装配（main 入口）。
func RegisterAll(r *gin.Engine, db *gorm.DB, cfg *config.Config) {
	r.Use(middleware.RequestID(), middleware.Recover())
	RegisterAuthRoutes(r, db, cfg)
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"code": 40004, "msg": "not found"})
	})
}
