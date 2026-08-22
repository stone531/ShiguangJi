// Package router 路由注册。
package router

import (
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
	commentSvc       *service.CommentService
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
	if commentSvc != nil {
		commentSvc.Close()
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

		// 后台写操作全部要求 XHR 头（CSRF 纵深防御，配合 SameSite）
		protected := admin.Group("", middleware.JWT(cfg), middleware.RequireXHR())
		protected.POST("/password", authH.ChangePassword)
		protected.GET("/profile", authH.GetProfile)
		protected.PUT("/profile", authH.UpdateProfile)
		protected.GET("/stats", h.NewStatsHandler(service.NewStatsService(db)).Get)

		// 文章管理（admin 路径下 :id，与前台 :slug 不同组，无 Gin 路由冲突）
		postSvc := service.NewPostService(db)
		postH := h.NewPostHandler(postSvc)
		protected.GET("/posts", postH.ListAdmin)
		protected.POST("/posts", postH.Create)
		protected.GET("/posts/:id", postH.GetByID)
		protected.PUT("/posts/:id", postH.Update)
		protected.PUT("/posts/:id/status", postH.SetStatus)
		protected.DELETE("/posts/:id", postH.Delete)

		// 前台公开（M3：列表筛选/详情/归档/站点信息/分类标签）
		api.GET("/posts", postH.ListPublic)
		api.GET("/posts/:slug", postH.GetBySlugPublic)
		api.GET("/archive", postH.Archive)
		api.GET("/site", h.NewSiteHandler(service.NewSiteService(db)).GetPublic)
		taxH := h.NewTaxonomyHandler(service.NewTaxonomyService(db))
		api.GET("/categories", taxH.ListCategories)
		api.GET("/tags", taxH.ListTags)

		// 评论（M4：公开读写 + 后台管理；写请求要求 XHR 头防 CSRF）
		if commentSvc != nil {
			commentSvc.Close() // 测试反复装配时释放上一个实例的限流器 goroutine（每个实例只被关闭一次）
		}
		commentSvc = service.NewCommentService(db)
		commentH := h.NewCommentHandler(commentSvc)
		api.GET("/posts/:slug/comments", commentH.ListPublic)
		api.POST("/posts/:slug/comments", middleware.RequireXHR(), commentH.Create)
		protected.GET("/comments", commentH.ListAdmin)
		protected.PUT("/comments/:id/review", commentH.Review)
		protected.POST("/comments/:id/reply", commentH.Reply)
		protected.DELETE("/comments/:id", commentH.Delete)

		// 点赞（M4：指纹匿名标识，POST 要求 XHR 头）
		likeH := h.NewLikeHandler(service.NewLikeService(db, cfg.LikeSalt))
		api.GET("/posts/:slug/like", likeH.Status)
		api.POST("/posts/:slug/like", middleware.RequireXHR(), likeH.Toggle)

		// 站点设置 / 分类标签管理（M4）
		siteH := h.NewSiteHandler(service.NewSiteService(db))
		protected.GET("/site", siteH.AdminGet)
		protected.PUT("/site", siteH.AdminSave)
		protected.GET("/categories", taxH.ListCategoriesAdmin)
		protected.POST("/categories", taxH.SaveCategory)
		protected.DELETE("/categories/:id", taxH.DeleteCategory)
		protected.GET("/tags", taxH.ListTagsAdmin)
		protected.POST("/tags", taxH.SaveTag)
		protected.DELETE("/tags/:id", taxH.DeleteTag)
		protected.POST("/tags/:id/merge", taxH.MergeTag)

		// 图片上传（含同步缩略图）
		uploadH := h.NewUploadHandler(service.NewUploadService(cfg.UploadDir, 5<<20))
		protected.POST("/upload", uploadH.Upload)
	}
}

// RegisterAll 完整装配（main 入口）。
func RegisterAll(r *gin.Engine, db *gorm.DB, cfg *config.Config) {
	r.Use(middleware.RequestID(), middleware.Recover())
	RegisterAuthRoutes(r, db, cfg)
	r.NoRoute(newSPAStatic(db, cfg).handle)
}
