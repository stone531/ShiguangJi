// Package config 加载 .env 与环境变量，进程内单例。
package config

import (
	"fmt"
	"log"
	"net/url"
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
		dsn := getEnv("DSN", "root:root@tcp(127.0.0.1:3306)/personal_blog?collation=utf8mb4_unicode_ci&parseTime=True&loc=Local")
		if user := os.Getenv("DB_USER"); user != "" {
			// 简单形式：离散 DB_* 变量优先于 DSN，免拼连接串
			dsn = buildDSN(getEnv("DB_HOST", "127.0.0.1"), getEnv("DB_PORT", "3306"),
				user, os.Getenv("DB_PASSWORD"), getEnv("DB_NAME", "personal_blog"))
		}
		global = &Config{
			Addr:       getEnv("ADDR", ":8080"),
			DSN:        dsn,
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

// buildDSN 由离散变量组装 MySQL DSN（go-sql-driver 格式）；
// 密码可能含 @ : / 等特殊字符，按驱动要求做 URL 转义。
func buildDSN(host, port, user, pass, name string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?collation=utf8mb4_unicode_ci&parseTime=True&loc=Local",
		url.QueryEscape(user), url.QueryEscape(pass), host, port, name)
}
