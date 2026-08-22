// Package config 加载 .env 与环境变量，进程内单例。
package config

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

// Config 应用配置。
type Config struct {
	Addr       string
	DSN        string
	JWTSecret  string
	JWTTimeout time.Duration
	UploadDir  string
	WebDist    string
	LikeSalt   string
	IsProd     bool
}

var (
	once   sync.Once
	global *Config
)

// Load 进程单例加载；测试中可调用 Reset。
func Load() *Config {
	once.Do(func() {
		_ = godotenv.Load() // .env 不存在时用真实环境变量
		global = &Config{
			Addr:       getEnv("ADDR", ":8080"),
			DSN:        getEnv("DSN", "root:root@tcp(127.0.0.1:3306)/personal_blog?charset=utf8mb4&parseTime=True&loc=Local"),
			JWTSecret:  getEnv("JWT_SECRET", ""),
			JWTTimeout: 2 * time.Hour,
			UploadDir:  getEnv("UPLOAD_DIR", "./uploads"),
			WebDist:    getEnv("WEB_DIST", "./dist"),
			LikeSalt:   getEnv("LIKE_SALT", ""),
			IsProd:     os.Getenv("APP_ENV") == "prod",
		}
		if global.JWTSecret == "" {
			log.Fatal("JWT_SECRET 未配置") // 启动期失败优于运行期裸奔
		}
	})
	return global
}

// Reset 仅用于测试重载配置，非并发安全。
func Reset() { once = sync.Once{}; global = nil }

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
