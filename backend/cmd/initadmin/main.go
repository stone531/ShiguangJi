// cmd/initadmin/main.go — 创建/重置管理员账号并应用数据库迁移。
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"shiguangji/internal/config"
	"shiguangji/internal/dao"
	"shiguangji/internal/model"
)

// 用法: go run ./cmd/initadmin -username admin -password <8+位> [-nickname 博主]
// 幂等：账号存在则重置密码并置 must_change_password=true。
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
	if res.RowsAffected == 0 { // 已存在 → 重置密码
		if err := db.WithContext(ctx).Model(&admin).Updates(map[string]interface{}{
			"password_hash": string(hash), "must_change_password": true,
		}).Error; err != nil {
			log.Fatal(err)
		}
	}
	fmt.Println("admin ready:", *username)
}

// applyMigrations 读取 migrations/*.sql 按文件名排序逐条执行（幂等语句）。
// 001_init.sql 含多个语句：按分号拆分逐条 Exec（本迁移字符串字面量不含分号，可安全拆分），
// 避免 MySQL DSN 追加 multiStatements=true 的兼容性问题。
func applyMigrations(ctx context.Context, db *gorm.DB) error {
	entries, err := os.ReadDir("migrations")
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".sql" {
			continue
		}
		sqlBytes, err := os.ReadFile(filepath.Join("migrations", e.Name()))
		if err != nil {
			return err
		}
		for _, stmt := range splitStatements(string(sqlBytes)) {
			if err := db.WithContext(ctx).Exec(stmt).Error; err != nil {
				return fmt.Errorf("%s: %w", e.Name(), err)
			}
		}
		log.Println("applied", e.Name())
	}
	return nil
}

// splitStatements 按分号拆分多条 SQL；剔除行注释与空白后过滤空片段。
func splitStatements(sqlText string) []string {
	var stmts []string
	for _, stmt := range strings.Split(sqlText, ";") {
		var lines []string
		for _, line := range strings.Split(stmt, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "--") {
				continue
			}
			lines = append(lines, line)
		}
		if len(lines) == 0 {
			continue
		}
		if s := strings.TrimSpace(strings.Join(lines, "\n")); s != "" {
			stmts = append(stmts, s)
		}
	}
	return stmts
}
