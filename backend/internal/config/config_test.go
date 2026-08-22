package config

import (
	"os"
	"testing"
)

// TestBuildDSN 校验离散变量组装的 DSN 格式与特殊字符密码转义。
func TestBuildDSN(t *testing.T) {
	got := buildDSN("127.0.0.1", "3306", "blog", "p@ss/w:rd", "personal_blog")
	want := "blog:p%40ss%2Fw%3Ard@tcp(127.0.0.1:3306)/personal_blog?collation=utf8mb4_unicode_ci&parseTime=True&loc=Local"
	if got != want {
		t.Fatalf("buildDSN() = %q, want %q", got, want)
	}
}

// TestLoadDBUserPreferredOverDSN DB_USER 设置时离散变量优先，DSN 被忽略。
func TestLoadDBUserPreferredOverDSN(t *testing.T) {
	Reset()
	defer Reset()
	t.Setenv("JWT_SECRET", "unit-test-secret")
	t.Setenv("DSN", "dsnuser:dnspass@tcp(1.2.3.4:3307)/other?charset=utf8mb4")
	t.Setenv("DB_USER", "blog")
	t.Setenv("DB_PASSWORD", "secret")

	cfg := Load()
	want := "blog:secret@tcp(127.0.0.1:3306)/personal_blog?collation=utf8mb4_unicode_ci&parseTime=True&loc=Local"
	if cfg.DSN != want {
		t.Fatalf("DSN = %q, want %q", cfg.DSN, want)
	}
}

// TestLoadFallsBackToDSN 未设 DB_USER 时回落到 DSN 环境变量（向后兼容）。
func TestLoadFallsBackToDSN(t *testing.T) {
	Reset()
	defer Reset()
	t.Setenv("JWT_SECRET", "unit-test-secret")
	t.Setenv("DSN", "u:p@tcp(1.2.3.4:3307)/other?charset=utf8mb4")
	os.Unsetenv("DB_USER")

	cfg := Load()
	if cfg.DSN != "u:p@tcp(1.2.3.4:3307)/other?charset=utf8mb4" {
		t.Fatalf("DSN = %q, want DSN 环境变量原样生效", cfg.DSN)
	}
}
