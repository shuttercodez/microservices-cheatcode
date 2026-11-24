package auth

import (
	"testing"
	"time"
)

func TestGenerateAndVerifyJWT(t *testing.T) {
	secret := "super-secret-jwt-key-for-unit-test"

	t.Run("Success - generates valid HS256 JWT and verifies claims", func(t *testing.T) {
		token, err := GenerateJWT(secret, "admin", "admin", "inventory:read inventory:write", 10*time.Minute)
		if err != nil {
			t.Fatalf("unexpected error generating JWT: %v", err)
		}

		claims, err := VerifyJWT(token, secret)
		if err != nil {
			t.Fatalf("unexpected error verifying JWT: %v", err)
		}
		if claims.Sub != "admin" || claims.Role != "admin" {
			t.Errorf("unexpected claims: %+v", claims)
		}
	})

	t.Run("Fail - rejects JWT signed with wrong secret", func(t *testing.T) {
		token, _ := GenerateJWT(secret, "admin", "admin", "inventory:read", 10*time.Minute)
		if _, err := VerifyJWT(token, "wrong-secret"); err == nil {
			t.Errorf("expected signature verification error, got nil")
		}
	})

	t.Run("Fail - rejects expired JWT", func(t *testing.T) {
		token, _ := GenerateJWT(secret, "admin", "admin", "inventory:read", -5*time.Second)
		if _, err := VerifyJWT(token, secret); err == nil {
			t.Errorf("expected expired token error, got nil")
		}
	})
}
