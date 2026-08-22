package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"shiguangji/internal/config"
	"shiguangji/internal/dao"
	"shiguangji/internal/router"
)

func main() {
	cfg := config.Load()
	if cfg.IsProd {
		gin.SetMode(gin.ReleaseMode)
	}
	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Fatal("create upload dir: ", err)
	}
	db, err := dao.Open(cfg)
	if err != nil {
		log.Fatal(err)
	}

	r := gin.New()
	// 仅信任本机 nginx 反代：否则 gin 默认信任所有代理，X-Forwarded-For 可伪造，
	// 登录/评论限流与点赞指纹（均基于 ClientIP）全部可被绕过
	if err := r.SetTrustedProxies([]string{"127.0.0.1", "::1"}); err != nil {
		log.Fatal("set trusted proxies: ", err)
	}
	router.RegisterAll(r, db, cfg)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second, // slowloris 防护
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	log.Println("server listening on", cfg.Addr)

	// 优雅关停：等待信号 → 5 秒排水 → 关限流器 → 关 DB
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	router.CloseAuthLimiter()
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
	log.Println("bye")
}
