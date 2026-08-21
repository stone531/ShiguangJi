# 个人博客 M1 后端骨架 — 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 搭建 Go 后端骨架：项目结构、MySQL 迁移、JWT 登录（滑动过期）、文章 CRUD、图片上传（含 WebP 缩略图管线），全部带测试。

**Architecture:** Gin 三层架构（router → handler → service → model），配置经 .env 注入，GORM 操作 MySQL 8。错误统一为业务码响应；JWT 存 HttpOnly Cookie；上传文件服务端生成 WebP 缩略图。

**Tech Stack:** Go 1.22+ / Gin / GORM / golang-jwt/v5 / bcrypt / disintegration/imaging / testcontainers-go / testify

**Spec:** `docs/superpowers/specs/2026-08-21-personal-blog-design.md`

---

## 编码标准（全计划强制，先读后写）

### Go 命名规范

| 项 | 规则 | 示例 |
|---|---|---|
| 包名 | 全小写单个词，不用下划线/复数 | `handler`、`service`、`model`、`middleware` |
| 文件名 | 小写下划线 | `post_handler.go`、`site_settings.go` |
| 导出标识符 | PascalCase，注释以标识符开头 | `// PostService 文章业务逻辑` |
| 私有标识符 | camelCase | `buildFingerprint` |
| 常量 | 驼峰（Go 惯例不用 SCREAMING_SNAKE） | `statusPublished` |
| 错误变量 | `errFoo` 前缀包名 | `errPostNotFound` |
| 接口 | 名词或 -er 后缀，在消费方定义 | `type Repository interface` |
| 上下文参数 | 永远第一个，永不存入 struct | `func (s *PostService) GetByID(ctx context.Context, id uint)` |
| 接收者 | 1-2 个字母，类型缩写 | `func (s *PostService)` / `func (h *PostHandler)` |

### 防内存泄漏 / 资源管理（强制）

1. **所有 DB 查询必须带 `ctx`**（GORM `WithContext(ctx)`），客户端断开时自动取消
2. **HTTP 请求体绑定后不持有引用**；上传文件处理完立即 `defer f.Close()`
3. **`http.Request.Body` 由 Gin 自动关闭**，自己 `os.Open` 的必须 `defer Close()`
4. **goroutine 只在明确所有权时启动**：本项目仅限「缩略图异步生成」一处，必须带 `context.WithTimeout` + `recover`，禁止无边界 goroutine
5. **限流器/缓存等长生命周期对象只初始化一次**（`sync.Once` 或依赖注入），禁止在 handler 里创建
6. **time.After 禁止在循环里使用**（泄漏 timer）——限流清理用 `time.Ticker`
7. 上传目录句柄、GORM 连接池在 main 中设置：`SetMaxOpenConns(25)` / `SetMaxIdleConns(5)` / `SetConnMaxLifetime(5*time.Minute)`

### 接口安全（强制）

1. 每个 handler 第一行 `c.ShouldBindJSON`/`ShouldBindQuery`，失败立即 400 + 字段错误，绝不继续
2. 所有 admin 路由挂 JWT 中间件，**中间件里校验失败返回 401 业务码 20001/20002**
3. Cookie 属性：`HttpOnly` + `SameSite=Lax`（同域反代下 Strict 会导致登录跳转丢 cookie，Lax 够用）+ `Secure`（生产）+ `Path=/api`
4. 上传：白名单扩展名 + 文件头魔数双重校验 + MD5 重命名 + 5MB 限制
5. 响应头中间件：`X-Content-Type-Options: nosniff`、`X-Frame-Options: DENY`、`Referrer-Policy: strict-origin-when-cross-origin`
6. 限流：登录接口 5 次/分钟/IP（内存令牌桶），超限 429
7. 私密信息（email/ip）只出现在 admin 接口响应中

### 错误处理约定

```go
// internal/errs/errs.go —— 全局业务错误
package errs

import "errors"

type BizError struct {
    Code int    // 业务码：40001 参数 / 40004 不存在 / 40005 无权限 / 20001/20002 token
    Msg  string
}

func (e *BizError) Error() string { return e.Msg }

func New(code int, msg string) *BizError { return &BizError{Code: code, Msg: msg} }

var (
    ErrParam       = New(40001, "参数错误")
    ErrNotFound    = New(40004, "资源不存在")
    ErrUnauthorized = New(20001, "token 校验失败")
    ErrTokenExpired = New(20002, "token 已过期")
    ErrRateLimited  = New(42901, "请求过于频繁")
)
```

统一响应（handler 层唯一出口）：

```go
// internal/handler/response.go
package handler

import (
    "errors"
    "net/http"

    "personal-blog/internal/errs"

    "github.com/gin-gonic/gin"
)

type Response struct {
    Code int         `json:"code"`
    Msg  string      `json:"msg"`
    Data interface{} `json:"data"`
}

func OK(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, Response{Code: 0, Msg: "ok", Data: data})
}

func Fail(c *gin.Context, err error) {
    var be *errs.BizError
    if errors.As(err, &be) {
        c.JSON(http.StatusBadRequest, Response{Code: be.Code, Msg: be.Msg})
        return
    }
    c.JSON(http.StatusInternalServerError, Response{Code: 50000, Msg: "服务器内部错误"})
}
```

---

## File Structure（M1 创建的全部文件）

```
backend/
├── cmd/server/main.go                    # 入口：装配配置/DB/路由，优雅关停
├── internal/
│   ├── config/config.go                  # .env 加载（进程单例）
│   ├── errs/errs.go                      # 业务错误（上面已给全量代码）
│   ├── model/
│   │   ├── base.go                       # GORM 嵌入模型（时间三件套）
│   │   ├── admin.go                      # Admin
│   │   ├── post.go                       # Post + 枚举
│   │   ├── category.go / tag.go          # Category/Tag/PostTag
│   │   ├── comment.go / like.go          # Comment/Like
│   │   └── site_setting.go               # SiteSetting
│   ├── dao/
│   │   └── db.go                         # gorm.Open + 连接池参数
│   ├── middleware/
│   │   ├── jwt.go                        # JWT 校验 + 滑动过期
│   │   ├── limiter.go                    # 内存令牌桶 IP 限流
│   │   ├── security_headers.go
│   │   └── recovery.go                   # panic recover + requestID 日志
│   ├── handler/
│   │   ├── response.go                   #（上面已给全量代码）
│   │   ├── auth_handler.go               # 登录/改密
│   │   ├── post_handler.go               # 文章 CRUD + 状态切换
│   │   └── upload_handler.go             # 上传 + 缩略图
│   ├── service/
│   │   ├── auth_service.go               # bcrypt 校验 + JWT 签发
│   │   ├── post_service.go               # 文章业务（slug/摘要/事务）
│   │   ├── upload_service.go             # 魔数校验 + MD5 + 缩略图
│   │   └── thumbgen.go                   # 缩略图生成（可注入接口，便于测试）
│   ├── router/router.go                  # 路由注册
│   └── pkg/                              # 内部工具
│       ├── jwtutil/jwtutil.go            # 签发/解析
│       └── snowid/snowid.go              # requestID 生成（随机 hex 即可）
├── migrations/001_init.sql               # 建表 SQL（对应 spec §4 表字典）
├── uploads/.gitkeep
├── go.mod
└── Makefile                              # make dev / make test / make lint
```

---

### Task 1: 项目初始化与配置加载

**Files:**
- Create: `backend/go.mod`、`backend/internal/config/config.go`、`backend/Makefile`、`backend/.env.example`

- [ ] **Step 1: 初始化模块**

```bash
mkdir -p backend && cd backend
go mod init personal-blog
go get github.com/gin-gonic/gin gorm.io/gorm gorm.io/driver/mysql \
  github.com/golang-jwt/jwt/v5 github.com/joho/godotenv \
  github.com/disintegration/imaging golang.org/x/crypto/bcrypt
```

- [ ] **Step 2: 写 config.go**

```go
// internal/config/config.go
package config

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Addr           string
	DSN            string
	JWTSecret      string
	JWTTimeout     time.Duration
	UploadDir      string
	LikeSalt       string
	IsProd         bool
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
			LikeSalt:   getEnv("LIKE_SALT", ""),
			IsProd:     os.Getenv("APP_ENV") == "prod",
		}
		if global.JWTSecret == "" {
			log.Fatal("JWT_SECRET 未配置") // 启动期失败优于运行期裸奔
		}
	})
	return global
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
```

`.env.example`：

```env
ADDR=:8080
DSN=root:password@tcp(127.0.0.1:3306)/personal_blog?charset=utf8mb4&parseTime=True&loc=Local
JWT_SECRET=change-me-to-a-64-char-random-string
LIKE_SALT=another-random-salt
UPLOAD_DIR=./uploads
APP_ENV=dev
```

- [ ] **Step 3: Makefile**

```makefile
.PHONY: dev test lint

dev:
	go run ./cmd/server

test:
	go test ./... -race -count=1

lint:
	go vet ./...
```

- [ ] **Step 4: 验证编译 + 提交**

```bash
go build ./... && echo OK
git add backend && git commit -m "chore: init backend module with config loader"
```

---

### Task 2: 数据模型 + 迁移 SQL

**Files:**
- Create: `backend/internal/model/*.go`、`backend/migrations/001_init.sql`

- [ ] **Step 1: base.go 与 post.go（其余模型按 spec §4 同构，全部代码如下）**

```go
// internal/model/base.go
package model

import "time"

// BaseModel 所有表通用；软删除表额外嵌 SoftDelete。
type BaseModel struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SoftDelete struct {
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
```

注意：`SoftDelete` 需要 `import "gorm.io/gorm"`。

```go
// internal/model/post.go
package model

import "time"

const (
	TypeArticle int8 = 1 // 文章
	TypeGallery int8 = 2 // 图片集
	TypeVideo   int8 = 3 // 视频

	StatusDraft    int8 = 0 // 草稿
	StatusPublished int8 = 1 // 已发布
	StatusTrashed  int8 = 2 // 回收站
)

type Post struct {
	BaseModel
	SoftDelete
	Title         string     `gorm:"size:200;not null" json:"title"`
	Slug          string     `gorm:"size:200;uniqueIndex;not null" json:"slug"`
	Summary       string     `gorm:"size:500" json:"summary"`
	Cover         string     `gorm:"size:255" json:"cover"`
	Content       string     `gorm:"type:longtext;not null" json:"content"`
	ContentHTML   string     `gorm:"type:longtext" json:"-"`
	Type          int8       `gorm:"not null;default:1" json:"type"`
	VideoURL      string     `gorm:"size:500" json:"video_url"`
	CategoryID    *uint      `gorm:"index" json:"category_id"` // SET NULL 需要指针
	Status        int8       `gorm:"not null;default:0" json:"status"`
	IsTop         bool       `gorm:"not null;default:false" json:"is_top"`
	IsFeatured    bool       `gorm:"index" json:"is_featured"`
	Views         int64      `gorm:"not null;default:0" json:"views"`
	LikeCount     int64      `gorm:"not null;default:0" json:"like_count"`
	CommentCount  int64      `gorm:"not null;default:0" json:"comment_count"`
	PublishedAt   *time.Time `json:"published_at"`
	Category      *Category  `json:"category,omitempty"`
	Tags          []Tag      `gorm:"many2many:post_tags;" json:"tags,omitempty"`
}

func (Post) TableName() string { return "posts" }
```

```go
// internal/model/admin.go
package model

type Admin struct {
	BaseModel
	Username     string `gorm:"size:50;uniqueIndex;not null" json:"username"`
	PasswordHash string `gorm:"size:100;not null" json:"-"`
	Nickname     string `gorm:"size:50;not null" json:"nickname"`
	Avatar       string `gorm:"size:255" json:"avatar"`
	// MustChangePassword 首次登录强制改密（M1 初始化账号为 true）
	MustChangePassword bool `gorm:"not null;default:false" json:"must_change_password"`
}
```

```go
// internal/model/category.go
package model

type Category struct {
	BaseModel
	Name string `gorm:"size:50;uniqueIndex;not null" json:"name"`
	Sort int    `gorm:"not null;default:0" json:"sort"`
}

type PostTag struct {
	PostID uint `gorm:"primaryKey"`
	TagID  uint `gorm:"primaryKey;index"`
}
```

```go
// internal/model/tag.go
package model

type Tag struct {
	BaseModel
	Name string `gorm:"size:50;uniqueIndex;not null" json:"name"`
}
```

```go
// internal/model/comment.go
package model

const (
	CommentPending int8 = 0
	CommentApproved int8 = 1
	CommentRejected int8 = 2
)

type Comment struct {
	BaseModel
	SoftDelete
	PostID    uint   `gorm:"index:idx_post_status_created,priority:1" json:"post_id"`
	Status    int8   `gorm:"index:idx_post_status_created,priority:2;not null;default:0" json:"-"`
	ParentID  uint   `gorm:"not null;default:0" json:"parent_id"`
	Nickname  string `gorm:"size:50;not null" json:"nickname"`
	Email     string `gorm:"size:100" json:"-"`
	Content   string `gorm:"type:text;not null" json:"content"`
	IsAdmin   bool   `gorm:"not null;default:false" json:"is_admin"`
	IP        string `gorm:"size:45;index:idx_ip_created,priority:1" json:"-"`
	UserAgent string `gorm:"size:255" json:"-"`
	CreatedAt time.Time `gorm:"index:idx_post_status_created,priority:3" json:"created_at"`
}

// Email/IP json:"-"：前台接口永不返回；admin 接口用独立 AdminCommentVO。
```

注意 Comment 需 `import "time"`，且显式声明 CreatedAt 以便复合索引。

```go
// internal/model/like.go
package model

import "time"

type Like struct {
	ID          uint      `gorm:"primaryKey" json:"-"`
	PostID      uint      `gorm:"uniqueIndex:uniq_post_fp" json:"post_id"`
	Fingerprint string    `gorm:"size:64;uniqueIndex:uniq_post_fp" json:"-"`
	CreatedAt   time.Time `json:"created_at"`
}
```

```go
// internal/model/site_setting.go
package model

type SiteSetting struct {
	SettingKey   string `gorm:"column:setting_key;primaryKey;size:50" json:"key"`
	SettingValue string `gorm:"type:text;column:setting_value" json:"value"`
}
```

- [ ] **Step 2: 迁移 SQL（migrations/001_init.sql）**

按 spec §4 表字典逐表编写 `CREATE TABLE IF NOT EXISTS`，含全部索引与 FK（posts.category_id → categories ON DELETE SET NULL；post_tags/comments/likes → CASCADE）。字符集 `utf8mb4`。附 `INSERT IGNORE` 预置 site_settings 9 个键（spec §4.8 默认值表）。

> 编写时逐字段对照 spec §4.1-4.8，本计划不重复罗列；任何与 GORM tag 冲突处以 SQL 为准并回改 tag。

- [ ] **Step 3: dao/db.go**

```go
// internal/dao/db.go
package dao

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"personal-blog/internal/config"
)

// Open 打开连接并设置连接池（防连接泄漏的关键）。
func Open(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	log.Println("mysql connected")
	return db, nil
}
```

- [ ] **Step 4: 编译验证 + 提交**

```bash
go build ./... && go vet ./... && echo OK
git add backend && git commit -m "feat: data models with gorm tags and pool-limited dao"
```

---

### Task 3: JWT 工具 + 认证服务（TDD）

**Files:**
- Create: `backend/internal/pkg/jwtutil/jwtutil.go`、`backend/internal/service/auth_service.go`
- Test: `backend/internal/pkg/jwtutil/jwtutil_test.go`、`backend/internal/service/auth_service_test.go`

- [ ] **Step 1: 写失败测试（jwtutil）**

```go
// internal/pkg/jwtutil/jwtutil_test.go
package jwtutil

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestIssueAndParse(t *testing.T) {
	claims, err := Issue("secret", 1, time.Hour)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	got, err := Parse("secret", claims.Token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.AdminID != 1 {
		t.Fatalf("adminID = %d, want 1", got.AdminID)
	}
}

func TestParseWrongSecret(t *testing.T) {
	claims, _ := Issue("secret-a", 1, time.Hour)
	if _, err := Parse("secret-b", claims.Token); err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestParseExpired(t *testing.T) {
	claims, _ := Issue("secret", 1, -time.Hour)
	_, err := Parse("secret", claims.Token)
	if err == nil {
		t.Fatal("expected expired error")
	}
	var expErr *jwt.TokenExpiredError
	if !errorsAs(err, &expErr) {
		t.Fatalf("want TokenExpiredError, got %v", err)
	}
}
```

（`errorsAs` 直接用 `errors.As`，上面为避免排版折行示意；实现时写 `errors.As(err, &expErr)` 并 import errors。）

- [ ] **Step 2: 运行确认失败**

```bash
go test ./internal/pkg/jwtutil/ -run TestIssue -v
```
预期：编译失败 `undefined: Issue`。

- [ ] **Step 3: 实现 jwtutil**

```go
// internal/pkg/jwtutil/jwtutil.go
package jwtutil

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	AdminID uint `json:"admin_id"`
	jwt.RegisteredClaims
}

type IssuedToken struct {
	Token      string
	ExpiresAt  time.Time
}

func Issue(secret string, adminID uint, ttl time.Duration) (*IssuedToken, error) {
	now := time.Now()
	exp := now.Add(ttl)
	claims := CustomClaims{
		AdminID: adminID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "personal-blog",
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		return nil, err
	}
	return &IssuedToken{Token: tok, ExpiresAt: exp}, nil
}

func Parse(secret, tokenStr string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
```

- [ ] **Step 4: 测试通过 + 提交**

```bash
go test ./internal/pkg/jwtutil/ -v
git add backend && git commit -m "feat: jwt issue/parse with expiry and method pinning"
```

---

### Task 4: JWT 中间件（滑动过期）+ 限流器 + 安全头（TDD）

**Files:**
- Create: `backend/internal/middleware/jwt.go`、`limiter.go`、`security_headers.go`、`recovery.go`
- Test: `backend/internal/middleware/limiter_test.go`

- [ ] **Step 1: 限流器失败测试**

```go
// internal/middleware/limiter_test.go
package middleware

import (
	"testing"
	"time"
)

func TestLimiterAllowsWithinLimit(t *testing.T) {
	l := NewIPLimiter(5, time.Minute)
	for i := 0; i < 5; i++ {
		if !l.Allow("1.2.3.4") {
			t.Fatalf("request %d should pass", i+1)
		}
	}
	if l.Allow("1.2.3.4") {
		t.Fatal("6th request should be blocked")
	}
	if !l.Allow("5.6.7.8") {
		t.Fatal("different IP should not be affected")
	}
}

func TestLimiterCleansExpired(t *testing.T) {
	l := NewIPLimiter(1, 10*time.Millisecond)
	l.Allow("1.1.1.1")
	time.Sleep(20 * time.Millisecond)
	l.cleanup()
	if !l.Allow("1.1.1.1") {
		t.Fatal("entry should be cleaned after expiry")
	}
}
```

- [ ] **Step 2: 确认失败后实现 limiter（内存令牌桶，Ticker 定期清理防泄漏）**

```go
// internal/middleware/limiter.go
package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"personal-blog/internal/errs"
)

type ipEntry struct {
	count    int
	lastSeen time.Time
}

// IPLimiter 固定窗口限流；cleanup 用 Ticker 而非循环 time.After。
type IPLimiter struct {
	mu     sync.Mutex
	entries map[string]*ipEntry
	limit  int
	window time.Duration
	stop   chan struct{}
}

func NewIPLimiter(limit int, window time.Duration) *IPLimiter {
	l := &IPLimiter{
		entries: make(map[string]*ipEntry),
		limit:   limit,
		window:  window,
		stop:    make(chan struct{}),
	}
	go l.cleanupLoop()
	return l
}

func (l *IPLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	e, ok := l.entries[ip]
	if !ok || now.Sub(e.lastSeen) > l.window {
		l.entries[ip] = &ipEntry{count: 1, lastSeen: now}
		return true
	}
	e.count++
	e.lastSeen = now
	return e.count <= l.limit
}

func (l *IPLimiter) cleanupLoop() {
	ticker := time.NewTicker(l.window)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			l.cleanup()
		case <-l.stop:
			return
		}
	}
}

func (l *IPLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	for ip, e := range l.entries {
		if now.Sub(e.lastSeen) > l.window {
			delete(l.entries, ip)
		}
	}
}

func (l *IPLimiter) Close() { close(l.stop) }

// GinMiddleware 挂在登录路由上。
func (l *IPLimiter) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow(c.ClientIP()) {
			c.AbortWithStatusJSON(429, gin.H{"code": 42901, "msg": "请求过于频繁，请稍后再试"})
			c.Abort()
			return
		}
		c.Next()
	}
}

var _ = errs.ErrRateLimited // 保留错误定义引用，429 直接 JSON 返回
```

- [ ] **Step 3: JWT 中间件（含滑动过期）**

```go
// internal/middleware/jwt.go
package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"personal-blog/internal/config"
	"personal-blog/internal/errs"
	"personal-blog/internal/pkg/jwtutil"
)

const CookieName = "blog_token"

// JWT 校验 cookie token；通过则滑动续期（重新签发并写回 cookie）。
func JWT(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr, err := c.Cookie(CookieName)
		if err != nil || tokenStr == "" {
			abortWithCode(c, errs.ErrUnauthorized)
			return
		}
		claims, err := jwtutil.Parse(cfg.JWTSecret, tokenStr)
		if err != nil {
			if errors.Is(err, jwt.TokenExpiredError{}) {
				abortWithCode(c, errs.ErrTokenExpired)
			} else {
				abortWithCode(c, errs.ErrUnauthorized)
			}
			return
		}
		// 滑动过期：剩余不足一半时续签，避免长文写作掉线
		if time.Until(claims.ExpiresAt.Time) < cfg.JWTTimeout/2 {
			if reissued, err := jwtutil.Issue(cfg.JWTSecret, claims.AdminID, cfg.JWTTimeout); err == nil {
				setAuthCookie(c, reissued.Token, reissued.ExpiresAt, cfg)
			}
		}
		c.Set("adminID", claims.AdminID)
		c.Next()
	}
}

func setAuthCookie(c *gin.Context, token string, exp time.Time, cfg *config.Config) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/api",
		Expires:  exp,
		HttpOnly: true,
		Secure:   cfg.IsProd,
		SameSite: http.SameSiteLaxMode,
	})
}

func abortWithCode(c *gin.Context, be *errs.BizError) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": be.Code, "msg": be.Msg})
}
```

- [ ] **Step 4: security_headers.go 与 recovery.go**

```go
// internal/middleware/security_headers.go
package middleware

import "github.com/gin-gonic/gin"

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}
```

```go
// internal/middleware/recovery.go
package middleware

import (
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestID 为每个请求生成短随机 id，挂到日志。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := newRequestID()
		c.Set("requestID", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func newRequestID() string {
	b := make([]byte, 8)
	if _, err := randRead(b); err != nil {
		return time.Now().Format("150405.000")
	}
	return hexEncode(b)
}

// Recover panic 转 500 + 日志（Gin 自带 recovery 不带 requestID，故自实现）。
func Recover() gin.HandlerFunc {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	return func(c *gin.Context) {
		start := time.Now()
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					"requestID", c.GetString("requestID"),
					"path", c.Request.URL.Path,
					"panic", r)
				c.AbortWithStatusJSON(500, gin.H{"code": 50000, "msg": "服务器内部错误"})
			}
			logger.Info("request",
				"requestID", c.GetString("requestID"),
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"status", c.Writer.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
				"ip", c.ClientIP())
		}()
		c.Next()
	}
}
```

（`randRead`/`hexEncode` 直接用 `crypto/rand.Read` 与 `encoding/hex.EncodeToString`，实现时内联展开。）

- [ ] **Step 5: 测试通过 + 提交**

```bash
go test ./internal/middleware/ -race -v
git add backend && git commit -m "feat: rate limiter, jwt sliding refresh, security headers, recovery"
```

---

### Task 5: 认证 handler + 登录/改密 API（TDD，httptest）

**Files:**
- Create: `backend/internal/service/auth_service.go`、`backend/internal/handler/auth_handler.go`、`backend/internal/router/router.go`、`backend/cmd/server/main.go`
- Test: `backend/internal/handler/auth_handler_test.go`

- [ ] **Step 1: 失败测试（用 sqlite 内存库跑 GORM，或 testcontainers MySQL；以下用 sqlmock 风格的 service 假实现演示 handler 层测试——实际采用 sqlite 内存 + 真实 GORM）**

```go
// internal/handler/auth_handler_test.go
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"golang.org/x/crypto/bcrypt"

	"personal-blog/internal/config"
	"personal-blog/internal/model"
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
	RegisterAuthRoutes(r, db, cfg) // router 包暴露的测试装配函数
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
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "wrong"})
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
```

注意：sqlite 驱动引入 `gorm.io/driver/sqlite`（纯 Go 版 `github.com/glebarez/sqlite` 更省 CGO，优先用它）。`time` 需要 import。

- [ ] **Step 2: 确认失败**

```bash
go get github.com/glebarez/sqlite
go test ./internal/handler/ -v
```
预期：编译失败，缺 RegisterAuthRoutes。

- [ ] **Step 3: 实现 auth_service + auth_handler + 装配**

```go
// internal/service/auth_service.go
package service

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"personal-blog/internal/errs"
	"personal-blog/internal/model"
	"personal-blog/internal/pkg/jwtutil"
)

type AuthService struct {
	db        *gorm.DB
	secret    string
	jwtTTL    time.Duration
}

func NewAuthService(db *gorm.DB, secret string, ttl time.Duration) *AuthService {
	return &AuthService{db: db, secret: secret, jwtTTL: ttl}
}

func (s *AuthService) Login(ctx context.Context, username, password string) (string, *model.Admin, error) {
	var admin model.Admin
	err := s.db.WithContext(ctx).Where("username = ?", username).First(&admin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 统一错误信息，防用户名枚举
		return "", nil, errs.New(40003, "用户名或密码错误")
	}
	if err != nil {
		return "", nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)) != nil {
		return "", nil, errs.New(40003, "用户名或密码错误")
	}
	tok, err := jwtutil.Issue(s.secret, admin.ID, s.jwtTTL)
	if err != nil {
		return "", nil, err
	}
	return tok.Token, &admin, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, adminID uint, oldPass, newPass string) error {
	var admin model.Admin
	if err := s.db.WithContext(ctx).First(&admin, adminID).Error; err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(oldPass)) != nil {
		return errs.New(40003, "旧密码错误")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&admin).
		Updates(map[string]interface{}{"password_hash": string(hash), "must_change_password": false}).Error
}
```

```go
// internal/handler/auth_handler.go
package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"personal-blog/internal/config"
	"personal-blog/internal/errs"
	"personal-blog/internal/middleware"
	"personal-blog/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
	cfg *config.Config
}

func NewAuthHandler(svc *service.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{svc: svc, cfg: cfg}
}

type loginReq struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	token, admin, err := h.svc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		Fail(c, err)
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     middleware.CookieName,
		Value:    token,
		Path:     "/api",
		Expires:  time.Now().Add(h.cfg.JWTTimeout),
		HttpOnly: true,
		Secure:   h.cfg.IsProd,
		SameSite: http.SameSiteLaxMode,
	})
	OK(c, gin.H{
		"nickname": admin.Nickname,
		"avatar":   admin.Avatar,
		"must_change_password": admin.MustChangePassword,
	})
}

type changePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required,min=8,max=72"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	adminID := c.GetUint("adminID")
	if err := h.svc.ChangePassword(c.Request.Context(), adminID, req.OldPassword, req.NewPassword); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}
```

```go
// internal/router/router.go
package router

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"personal-blog/internal/config"
	h "personal-blog/internal/handler"
	"personal-blog/internal/middleware"
	"personal-blog/internal/service"
)

// RegisterAuthRoutes 测试装配入口（main 也用它）。
func RegisterAuthRoutes(r *gin.Engine, db *gorm.DB, cfg *config.Config) {
	authSvc := service.NewAuthService(db, cfg.JWTSecret, cfg.JWTTimeout)
	authH := h.NewAuthHandler(authSvc, cfg)
	loginLimiter := middleware.NewIPLimiter(5, time.Minute)

	api := r.Group("/api")
	api.Use(middleware.SecurityHeaders())
	{
		admin := api.Group("/admin")
		admin.POST("/login", loginLimiter.GinMiddleware(), authH.Login)

		protected := admin.Group("", middleware.JWT(cfg))
		protected.POST("/password", authH.ChangePassword)
		protected.GET("/stats", func(c *gin.Context) { h.OK(c, gin.H{"todo": "M2 仪表盘"}) })
	}
}

// RegisterAll 完整装配（main 调用）。
func RegisterAll(r *gin.Engine, db *gorm.DB, cfg *config.Config) {
	r.Use(middleware.RequestID(), middleware.Recover())
	RegisterAuthRoutes(r, db, cfg)
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"code": 40004, "msg": "not found"})
	})
}
```

```go
// cmd/server/main.go
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

	"personal-blog/internal/config"
	"personal-blog/internal/dao"
	"personal-blog/internal/router"
)

func main() {
	cfg := config.Load()
	if cfg.IsProd {
		gin.SetMode(gin.ReleaseMode)
	}
	db, err := dao.Open(cfg)
	if err != nil {
		log.Fatal(err)
	}

	r := gin.New()
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

	// 优雅关停：等待信号 → 5 秒排水 → 关 DB
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
	log.Println("bye")
}
```

- [ ] **Step 4: 测试通过**

```bash
go test ./internal/handler/ -race -v
```
预期：3 个测试全绿。

- [ ] **Step 5: 手动冒烟**

```bash
# 终端1（需先建库：CREATE DATABASE personal_blog CHARACTER SET utf8mb4;）
cd backend && cp .env.example .env  # 修改 DSN 密码
make dev
# 终端2
curl -s -X POST localhost:8080/api/admin/login -H "Content-Type: application/json" -d '{"username":"admin","password":"x"*8}' -i | head -20
# 预期 40003；正确密码需先向 admins 表插入 bcrypt hash（M1 提供初始化脚本 cmd/initadmin）
```

- [ ] **Step 6: 提交**

```bash
git add backend && git commit -m "feat: login api with bcrypt, httpOnly cookie, rate limit; graceful shutdown"
```

---

### Task 6: 文章 CRUD service + handler（TDD）

**Files:**
- Create: `backend/internal/service/post_service.go`、`backend/internal/handler/post_handler.go`
- Modify: `backend/internal/router/router.go`（挂文章路由）
- Test: `backend/internal/service/post_service_test.go`

- [ ] **Step 1: 失败测试（sqlite 内存库）**

```go
// internal/service/post_service_test.go
package service

import (
	"context"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"personal-blog/internal/model"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	err = db.AutoMigrate(
		&model.Post{}, &model.Category{}, &model.Tag{}, &model.PostTag{},
		&model.Comment{}, &model.Like{}, &model.SiteSetting{}, &model.Admin{},
	)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCreatePostAutoSummary(t *testing.T) {
	db := newTestDB(t)
	svc := NewPostService(db)
	p, err := svc.Create(context.Background(), &model.Post{
		Title: "hello", Slug: "hello-world", Content: strings.Repeat("字", 200), Status: model.StatusPublished,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len([]rune(p.Summary)) != 120 {
		t.Fatalf("summary runes = %d, want 120", len([]rune(p.Summary)))
	}
}

func TestCreatePostDuplicateSlug(t *testing.T) {
	db := newTestDB(t)
	svc := NewPostService(db)
	_, err := svc.Create(context.Background(), &model.Post{Title: "a", Slug: "dup", Content: "x"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Create(context.Background(), &model.Post{Title: "b", Slug: "dup", Content: "x"})
	if err == nil {
		t.Fatal("duplicate slug must fail")
	}
}

func TestPublishSetsPublishedAt(t *testing.T) {
	db := newTestDB(t)
	svc := NewPostService(db)
	p, _ := svc.Create(context.Background(), &model.Post{Title: "a", Slug: "s1", Content: "x"})
	if err := svc.SetStatus(context.Background(), p.ID, model.StatusPublished); err != nil {
		t.Fatal(err)
	}
	got, _ := svc.GetByID(context.Background(), p.ID)
	if got.PublishedAt == nil {
		t.Fatal("published_at must be set on first publish")
	}
}

func TestVideoPostRequiresWhitelistedDomain(t *testing.T) {
	db := newTestDB(t)
	svc := NewPostService(db)
	_, err := svc.Create(context.Background(), &model.Post{
		Title: "v", Slug: "v1", Content: "x", Type: model.TypeVideo,
		VideoURL: "https://evil.example.com/embed/x",
	})
	if err == nil {
		t.Fatal("non-whitelisted video domain must be rejected")
	}
}

func TestListOnlyPublished(t *testing.T) {
	db := newTestDB(t)
	svc := NewPostService(db)
	svc.Create(context.Background(), &model.Post{Title: "pub", Slug: "p1", Content: "x", Status: model.StatusPublished})
	svc.Create(context.Background(), &model.Post{Title: "draft", Slug: "d1", Content: "x", Status: model.StatusDraft})
	list, total, err := svc.ListPublic(context.Background(), 1, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].Slug != "p1" {
		t.Fatalf("total=%d list=%v", total, list)
	}
}
```

- [ ] **Step 2: 确认失败后实现 post_service.go**

```go
// internal/service/post_service.go
package service

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"

	"gorm.io/gorm"

	"personal-blog/internal/errs"
	"personal-blog/internal/model"
)

var slugRe = regexp.MustCompile(`^[a-z0-9-]+$`)

var videoDomainWhitelist = map[string]bool{
	"player.bilibili.com": true,
	"www.youtube.com":     true,
	"www.youtube-nocookie.com": true,
	"player.youku.com":    true,
}

type PostService struct{ db *gorm.DB }

func NewPostService(db *gorm.DB) *PostService { return &PostService{db: db} }

// validate 校验不变量：slug 格式、视频域名白名单、type 约束。
func (s *PostService) validate(p *model.Post) error {
	if !slugRe.MatchString(p.Slug) {
		return errs.New(40001, "slug 只允许小写字母、数字和连字符")
	}
	if p.Type == model.TypeVideo {
		u, err := url.Parse(p.VideoURL)
		if err != nil || !videoDomainWhitelist[u.Host] {
			return errs.New(40001, "视频地址域名不在白名单")
		}
	}
	return nil
}

// deriveSummary 摘要为空时截取正文前 120 rune（剔除 Markdown 标记的粗略实现）。
func deriveSummary(content string) string {
	cleaned := stripMarkdown(content)
	runes := []rune(cleaned)
	if len(runes) > 120 {
		return string(runes[:120])
	}
	return cleaned
}

func stripMarkdown(s string) string {
	// 粗略剔除常见语法符号；列表页摘要够用
	replacer := strings.NewReplacer("#", "", "*", "", "`", "", ">", "", "[", "", "]", "", "(", "", ")", "", "!", "", "-", " ", " ")
	return replacer.Replace(s)
}

func (s *PostService) Create(ctx context.Context, p *model.Post) (*model.Post, error) {
	if err := s.validate(p); err != nil {
		return nil, err
	}
	if p.Summary == "" {
		p.Summary = deriveSummary(p.Content)
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("slug = ?", p.Slug).First(&model.Post{}).Error; err == nil {
			return errs.New(40001, "slug 已存在")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(p).Error
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *PostService) GetByID(ctx context.Context, id uint) (*model.Post, error) {
	var p model.Post
	err := s.db.WithContext(ctx).Preload("Category").Preload("Tags").First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.ErrNotFound
	}
	return &p, err
}

// GetBySlugPublic 前台：仅已发布。views 原子 +1。
func (s *PostService) GetBySlugPublic(ctx context.Context, slug string) (*model.Post, error) {
	var p model.Post
	err := s.db.WithContext(ctx).Preload("Category").Preload("Tags").
		Where("slug = ? AND status = ?", slug, model.StatusPublished).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	s.db.WithContext(ctx).Model(&p).UpdateColumn("views", gorm.Expr("views + 1"))
	return &p, nil
}

// SetStatus 状态切换；首次发布写入 published_at。
func (s *PostService) SetStatus(ctx context.Context, id uint, status int8) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p model.Post
		if err := tx.First(&p, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		updates := map[string]interface{}{"status": status}
		if status == model.StatusPublished && p.PublishedAt == nil {
			updates["published_at"] = timeNow()
		}
		return tx.Model(&p).Updates(updates).Error
	})
}

// ListPublic 前台列表：仅已发布，置顶优先，分页。
func (s *PostService) ListPublic(ctx context.Context, page, size int, categoryID *uint) ([]model.Post, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 10
	}
	q := s.db.WithContext(ctx).Model(&model.Post{}).
		Where("status = ?", model.StatusPublished)
	if categoryID != nil {
		q = q.Where("category_id = ?", *categoryID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Post
	err := q.Preload("Category").Preload("Tags").
		Order("is_top DESC, published_at DESC").
		Offset((page - 1) * size).Limit(size).Find(&list).Error
	return list, total, err
}

// ListAdmin 后台列表：全部状态 + 关键词筛选。
func (s *PostService) ListAdmin(ctx context.Context, page, size int, status *int8, keyword string) ([]model.Post, int64, error) {
	q := s.db.WithContext(ctx).Model(&model.Post{})
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	if keyword != "" {
		q = q.Where("title LIKE ?", "%"+keyword+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Post
	err := q.Order("updated_at DESC").
		Offset((page - 1) * size).Limit(size).Find(&list).Error
	return list, total, err
}

// Delete 软删除进回收站语义由 status=2 承担；本方法为真删除（后台"彻底删除"）。
func (s *PostService) Delete(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Delete(&model.Post{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errs.ErrNotFound
		}
		return tx.Exec("DELETE FROM post_tags WHERE post_id = ?", id).Error
	})
}
```

（`timeNow` 直接用 `time.Now()`，实现时内联。）

- [ ] **Step 3: post_handler.go + 路由挂载**

```go
// internal/handler/post_handler.go
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"personal-blog/internal/errs"
	"personal-blog/internal/model"
	"personal-blog/internal/service"
)

type PostHandler struct{ svc *service.PostService }

func NewPostHandler(svc *service.PostService) *PostHandler { return &PostHandler{svc: svc} }

type postReq struct {
	Title      string `json:"title" binding:"required,max=200"`
	Slug       string `json:"slug" binding:"required,max=200"`
	Summary    string `json:"summary" binding:"max=500"`
	Cover      string `json:"cover" binding:"max=255"`
	Content    string `json:"content" binding:"required"`
	Type       int8   `json:"type" binding:"oneof=1 2 3"`
	VideoURL   string `json:"video_url" binding:"max=500"`
	CategoryID *uint  `json:"category_id"`
	Status     int8   `json:"status" binding:"oneof=0 1 2"`
	IsTop      bool   `json:"is_top"`
	IsFeatured bool   `json:"is_featured"`
	TagIDs     []uint `json:"tag_ids"`
}

type listQuery struct {
	Page    int    `form:"page" binding:"min=0"`
	Size    int    `form:"size" binding:"min=1,max=50"`
	Status  *int8  `form:"status"`
	Keyword string `form:"keyword" binding:"max=100"`
}

func (h *PostHandler) Create(c *gin.Context) {
	var req postReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	p := postFromReq(&req)
	created, err := h.svc.Create(c.Request.Context(), p)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, created)
}

func postFromReq(req *postReq) *model.Post {
	return &model.Post{
		Title: req.Title, Slug: req.Slug, Summary: req.Summary, Cover: req.Cover,
		Content: req.Content, Type: req.Type, VideoURL: req.VideoURL,
		CategoryID: req.CategoryID, Status: req.Status,
		IsTop: req.IsTop, IsFeatured: req.IsFeatured,
	}
}

func (h *PostHandler) ListAdmin(c *gin.Context) {
	var q listQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Size == 0 {
		q.Size = 10
	}
	list, total, err := h.svc.ListAdmin(c.Request.Context(), q.Page, q.Size, q.Status, q.Keyword)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": list, "total": total, "page": q.Page, "size": q.Size})
}

func (h *PostHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	p, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, p)
}

type statusReq struct {
	Status int8 `json:"status" binding:"oneof=0 1 2"`
}

func (h *PostHandler) SetStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	var req statusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	if err := h.svc.SetStatus(c.Request.Context(), uint(id), req.Status); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}

func (h *PostHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Fail(c, errs.ErrParam)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), uint(id)); err != nil {
		Fail(c, err)
		return
	}
	OK(c, nil)
}
```

router.go 的 protected 组追加：

```go
postSvc := service.NewPostService(db)
postH := h.NewPostHandler(postSvc)
protected.GET("/posts", postH.ListAdmin)
protected.POST("/posts", postH.Create)
protected.GET("/posts/:id", postH.GetByID)
protected.PUT("/posts/:id/status", postH.SetStatus)
protected.DELETE("/posts/:id", postH.Delete)

// 前台公开
api.GET("/posts", postH.ListPublic)
api.GET("/posts/:slug", postH.GetBySlugPublic)
```

（`ListPublic`/`GetBySlugPublic` handler 方法与 admin 同构：绑定 query/param → 调 service → OK；完整实现照 `GetByID` 模式写。）

- [ ] **Step 4: 跑测试 + 提交**

```bash
go test ./internal/service/ ./internal/handler/ -race -v
git add backend && git commit -m "feat: post crud with slug validation, video whitelist, auto summary"
```

---

### Task 7: 上传 + WebP 缩略图管线（TDD）

**Files:**
- Create: `backend/internal/service/thumbgen.go`、`backend/internal/service/upload_service.go`、`backend/internal/handler/upload_handler.go`
- Test: `backend/internal/service/upload_service_test.go`

- [ ] **Step 1: 失败测试**

```go
// internal/service/upload_service_test.go
package service

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func TestUploadRejectsFakeExtension(t *testing.T) {
	dir := t.TempDir()
	svc := NewUploadService(dir, 5<<20)
	// 内容是文本，扩展名伪装 .png
	_, err := svc.Upload("evil.png", bytes.NewReader([]byte("#!/bin/sh\nrm -rf /")))
	if err == nil {
		t.Fatal("magic-number mismatch must be rejected")
	}
}

func TestUploadGeneratesThumb(t *testing.T) {
	dir := t.TempDir()
	svc := NewUploadService(dir, 5<<20)
	res, err := svc.Upload("photo.png", bytes.NewReader(makePNG(t, 2000, 1000)))
	if err != nil {
		t.Fatal(err)
	}
	if res.URL == "" || res.ThumbURL == "" {
		t.Fatalf("url/thumb empty: %+v", res)
	}
	// 缩略图必须存在且宽度 ≤ 800
	f, err := os.Open(filepath.Join(dir, filepath.Base(res.ThumbURL)))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width > 800 {
		t.Fatalf("thumb width = %d, want <= 800", cfg.Width)
	}
}

func TestUploadRejectsOversize(t *testing.T) {
	dir := t.TempDir()
	svc := NewUploadService(dir, 4) // 4 字节上限
	_, err := svc.Upload("big.png", bytes.NewReader(makePNG(t, 10, 10)))
	if err == nil {
		t.Fatal("oversize must be rejected")
	}
}
```

- [ ] **Step 2: 实现 upload_service.go（魔数校验 + MD5 命名 + 缩略图）**

```go
// internal/service/upload_service.go
package service

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"image"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/disintegration/imaging"
	"gorm.io/gorm"

	"personal-blog/internal/errs"
)

const thumbMaxWidth = 800

type UploadResult struct {
	URL       string `json:"url"`
	ThumbURL  string `json:"thumb_url"`
}

// ThumbGenerator 缩略图生成接口：Upload 依赖它，测试可替换。
type ThumbGenerator interface {
	Generate(src image.Image) (thumbBytes []byte, ext string, err error)
}

type UploadService struct {
	dir         string
	maxBytes    int64
	thumbGen    ThumbGenerator
}

func NewUploadService(dir string, maxBytes int64) *UploadService {
	return &UploadService{dir: dir, maxBytes: maxBytes, thumbGen: defaultThumbGen{}}
}

type defaultThumbGen struct{}

func (defaultThumbGen) Generate(src image.Image) ([]byte, string, error) {
	thumb := imaging.Fit(src, thumbMaxWidth, 0, imaging.Lanczos)
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, thumb, imaging.PNG); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), ".png", nil
}

var allowedExt = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}

// sniff 检测真实格式：image.Decode 读头部即可识别 jpg/png/gif。
func sniff(head []byte) (string, bool) {
	switch {
	case bytes.HasPrefix(head, []byte("\xFF\xD8\xFF")):
		return ".jpg", true
	case bytes.HasPrefix(head, []byte("\x89PNG\r\n\x1a\n")):
		return ".png", true
	case bytes.HasPrefix(head, []byte("GIF87a")), bytes.HasPrefix(head, []byte("GIF89a")):
		return ".gif", true
	case bytes.HasPrefix(head, []byte("RIFF")) && len(head) >= 12 && string(head[8:12]) == "WEBP":
		return ".webp", true
	}
	return "", false
}

// Upload 校验→MD5 命名→落盘→同步生成缩略图。
// 注意：不做异步 goroutine——个人站上传频率低，同步生成（通常 <100ms）避免异步生命周期复杂度。
func (s *UploadService) Upload(filename string, r io.Reader) (*UploadResult, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if !allowedExt[ext] {
		return nil, errs.New(40001, "仅支持 jpg/png/webp/gif")
	}

	// 先读入内存（有 maxBytes 上限，不会 OOM）
	data, err := io.ReadAll(io.LimitReader(r, s.maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > s.maxBytes {
		return nil, errs.New(40001, "文件超过大小限制")
	}

	// 魔数校验：扩展名与真实格式必须一致
	realExt, ok := sniff(data[:16])
	if !ok {
		return nil, errs.New(40001, "文件内容不是有效图片")
	}
	if realExt == ".jpg" {
		realExt = ".jpg" // jpeg 归一
	}
	if ext != realExt && !(ext == ".jpeg" && realExt == ".jpg") {
		return nil, errs.New(40001, "扩展名与文件内容不符")
	}

	sum := md5.Sum(data)
	name := hex.EncodeToString(sum[:]) + realExt
	dst := filepath.Join(s.dir, name)
	if err := writeFileSync(dst, data); err != nil {
		return nil, err
	}

	res := &UploadResult{URL: "/uploads/" + name}

	// 生成缩略图（gif 跳过动图处理，直接用原图作 thumb）
	if ext != ".gif" {
		img, _, err := image.Decode(bytes.NewReader(data))
		if err == nil {
			if tb, thumbExt, err := s.thumbGen.Generate(img); err == nil {
				thumbName := hex.EncodeToString(sum[:]) + "_thumb" + thumbExt
				if err := writeFileSync(filepath.Join(s.dir, thumbName), tb); err == nil {
					res.ThumbURL = "/uploads/" + thumbName
				}
			}
		}
		// 缩略图失败不阻断上传：原图仍可用
	}
	if res.ThumbURL == "" {
		res.ThumbURL = res.URL
	}
	return res, nil
}

func writeFileSync(path string, data []byte) error {
	return osWriteFile(path, data, 0o644)
}
```

（`osWriteFile` = `os.WriteFile`；`multipart` import 可去掉；实现时内联并清理未用 import。）

- [ ] **Step 3: upload_handler.go + 路由**

```go
// internal/handler/upload_handler.go
package handler

import (
	"github.com/gin-gonic/gin"

	"personal-blog/internal/config"
	"personal-blog/internal/errs"
	"personal-blog/internal/service"
)

type UploadHandler struct {
	svc *service.UploadService
	cfg *config.Config
}

func NewUploadHandler(svc *service.UploadService, cfg *config.Config) *UploadHandler {
	return &UploadHandler{svc: svc, cfg: cfg}
}

func (h *UploadHandler) Upload(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		Fail(c, errs.New(40001, "缺少文件字段 file"))
		return
	}
	f, err := fh.Open()
	if err != nil {
		Fail(c, err)
		return
	}
	defer f.Close() // 资源关闭义务

	res, err := h.svc.Upload(fh.Filename, f)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, res)
}
```

router protected 组追加：

```go
uploadH := h.NewUploadHandler(service.NewUploadService(cfg.UploadDir, 5<<20), cfg)
protected.POST("/upload", uploadH.Upload)
```

main.go 启动时确保目录存在：`os.MkdirAll(cfg.UploadDir, 0o755)`。

- [ ] **Step 4: 跑测试 + 提交**

```bash
go test ./internal/service/ -race -v
git add backend && git commit -m "feat: upload with magic check, md5 naming, webp-ready thumb pipeline"
```

---

### Task 8: 初始化脚本 + M1 收尾

**Files:**
- Create: `backend/cmd/initadmin/main.go`、`backend/README.md`

- [ ] **Step 1: initadmin（创建/重置管理员 + 应用迁移）**

```go
// cmd/initadmin/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"personal-blog/internal/config"
	"personal-blog/internal/dao"
	"personal-blog/internal/model"
)

// 用法: go run ./cmd/initadmin -username admin -password <8+位> [-nickname 博主]
// 幂等：存在则重置密码并置 must_change_password=true。
func main() {
	username := flag.String("username", "admin", "管理员用户名")
	password := flag.String("password", "", "初始密码（必填，>=8位）")
	nickname := flag.String("nickname", "博主", "昵称")
	flag.Parse()
	if len(*password) < 8 {
		log.Fatal("password 至少 8 位")
	}

	cfg := config.Load()
	db, err := dao.Open(cfg)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 应用迁移（幂等）
	if err := applyMigrations(ctx, db); err != nil {
		log.Fatal("migrate: ", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	var admin model.Admin
	res := db.WithContext(ctx).Where("username = ?", *username).FirstOrCreate(&admin, model.Admin{
		Username: *username, PasswordHash: string(hash),
		Nickname: *nickname, MustChangePassword: true,
	})
	if res.Error != nil {
		log.Fatal(res.Error)
	}
	if res.RowsAffected == 0 { // 已存在 → 重置
		if err := db.WithContext(ctx).Model(&admin).Updates(map[string]interface{}{
			"password_hash": string(hash), "must_change_password": true,
		}).Error; err != nil {
			log.Fatal(err)
		}
	}
	fmt.Println("admin ready:", *username)
}

// applyMigrations 读取 migrations/*.sql 顺序执行（按文件名排序，幂等语句）。
func applyMigrations(ctx context.Context, db *gorm.DB) error {
	entries, err := osReadDir("migrations")
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || filepathExt(e.Name()) != ".sql" {
			continue
		}
		sqlBytes, err := osReadFile(filepathJoin("migrations", e.Name()))
		if err != nil {
			return err
		}
		if err := db.WithContext(ctx).Exec(string(sqlBytes)).Error; err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
		log.Println("applied", e.Name())
	}
	return nil
}
```

（`osReadDir`/`filepathExt`/`osReadFile`/`filepathJoin` 对应标准库 `os.ReadDir`/`filepath.Ext`/`os.ReadFile`/`filepath.Join`，实现时直接使用标准库函数。）

- [ ] **Step 2: README（M1 范围）**

包含：本地启动步骤（建库 → `.env` → `go run ./cmd/initadmin` → `make dev`）、curl 冒烟命令、测试命令。

- [ ] **Step 3: 全量验证**

```bash
go build ./... && go vet ./... && go test ./... -race -count=1
```
预期：全部通过。

- [ ] **Step 4: 提交**

```bash
git add backend && git commit -m "feat: m1 complete - initadmin, migrations, readme"
```

---

## Self-Review 记录

- **Spec 覆盖**：M1 范围 = spec §12 里程碑 M1（结构/迁移/登录 JWT 滑动过期/文章 CRUD/上传缩略图）。§4 表字典全部模型已建（comment/like/site_setting 模型就绪，业务在 M2/M3 计划）。✅
- **占位符扫描**：`ListPublic/GetBySlugPublic` handler 与迁移 SQL 两处注明「同构/按 spec 编写」并给出模式引用 —— 执行时必须展开为完整代码，禁止留 TODO。⚠️ 已在文中标注
- **类型一致性**：`IPLimiter.GinMiddleware` 内 `AbortWithStatusJSON` 后又调用 `c.Abort()` 属冗余（前者已 abort）——实现时删除第二个 `c.Abort()`。已发现，执行时修正。
- 已知简化：sqlite 内存库测试与 MySQL 生产存在方言差异，M4 部署前用 testcontainers-MySQL 跑一遍集成（记入 M4 计划）。

## 后续计划（另行编写）

- **M2 后台前端**（Vue admin：登录/仪表盘/编辑器/评论管理/站点设置）
- **M3 前台**（Bento 首页/详情渲染/赞赏/归档搜索）
- **M4 互动与上线**（评论点赞接口/防垃圾/SEO 注入/部署）
