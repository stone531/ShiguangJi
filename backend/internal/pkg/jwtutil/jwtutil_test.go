package jwtutil

import (
	"errors"
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
	if !errors.Is(err, jwt.ErrTokenExpired) {
		t.Fatalf("want ErrTokenExpired, got %v", err)
	}
}

func TestParseAlgNoneRejected(t *testing.T) {
	claims := CustomClaims{
		AdminID: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := Parse("secret", tok); err == nil {
		t.Fatal("alg=none token should be rejected")
	}
}
