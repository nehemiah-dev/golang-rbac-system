package initializers

import (
	"os"
	"time"

	"github.com/Steve-s-Circle-on-System-Design/golang-rbac-system/internal/auth"
)

var JWT *auth.JWTUtil

func InitJWT() (*auth.JWTUtil, error) {
	accessTTL, err := time.ParseDuration(os.Getenv("JWT_ACCESS_TOKEN_TTL"))
	if err != nil {
		return nil, err
	}

	refreshTTL, err := time.ParseDuration(os.Getenv("JWT_REFRESH_TOKEN_TTL"))
	if err != nil {
		return nil, err
	}

	return auth.NewJWTUtil(auth.JWTConfig{
		AccessSecret:    os.Getenv("JWT_ACCESS_SECRET"),
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
		Issuer:          os.Getenv("JWT_ISSUER"),
	})
}
