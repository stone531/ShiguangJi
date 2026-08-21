// Package jwtutil 提供 JWT 签发与解析（HMAC 固定签名方法）。
package jwtutil

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims 携带 adminID 的自定义声明。
type CustomClaims struct {
	AdminID uint `json:"admin_id"`
	jwt.RegisteredClaims
}

// IssuedToken 签发结果：token 串与过期时间。
type IssuedToken struct {
	Token     string
	ExpiresAt time.Time
}

// Issue 签发 HMAC-SHA256 token。
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

// Parse 解析并校验 token；签名方法固定为 HMAC，防止算法替换攻击。
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
