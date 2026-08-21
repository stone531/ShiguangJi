package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"shiguangji/internal/config"
	"shiguangji/internal/errs"
	"shiguangji/internal/pkg/jwtutil"
)

// CookieName 认证 cookie 名。
const CookieName = "blog_token"

// JWT 校验 cookie token；通过则滑动续期（剩余不足一半时重新签发并写回 cookie）。
func JWT(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr, err := c.Cookie(CookieName)
		if err != nil || tokenStr == "" {
			abortWithCode(c, errs.ErrUnauthorized)
			return
		}
		claims, err := jwtutil.Parse(cfg.JWTSecret, tokenStr)
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				abortWithCode(c, errs.ErrTokenExpired)
			} else {
				abortWithCode(c, errs.ErrUnauthorized)
			}
			return
		}
		// 滑动过期：剩余不足一半时续签，避免长文写作掉线
		if claims.ExpiresAt != nil && time.Until(claims.ExpiresAt.Time) < cfg.JWTTimeout/2 {
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
