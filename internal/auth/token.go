package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrTokenExpired   = errors.New("token has expired")
	ErrTokenInvalid   = errors.New("token is invalid")
	ErrTokenMalformed = errors.New("token is malformed")
)

// claims inserted in a short lived access token
type AccessTokenClaims struct {
	jwt.RegisteredClaims

	Role   string    `json:"role"`
	UserID uuid.UUID `json:"uuid"`
}

// JWTConfig holds all config required to issue and validate tokesn
type JWTConfig struct {
	AccessSecret string
	Issuer       string

	AccessTokenTTL  time.Duration // 20 mins
	RefreshTokenTTL time.Duration // 7 days
}

type JWTUtil struct {
	cfg JWTConfig
}

// NewJWTUtil validates the config and returns a ready-to-use JWTUtil.
func NewJWTUtil(cfg JWTConfig) (*JWTUtil, error) {
	if cfg.AccessSecret == "" {
		return nil, errors.New("jwt: AccessSecret must not be empty")
	}
	if len(cfg.AccessSecret) < 32 {
		return nil, errors.New("jwt: AccessSecret must be at least 32 characters")
	}
	if cfg.AccessTokenTTL <= 0 {
		return nil, errors.New("jwt: AccessTokenTTL must be positive")
	}
	if cfg.RefreshTokenTTL <= 0 {
		return nil, errors.New("jwt: RefreshTokenTTL must be positive")
	}
	if cfg.Issuer == "" {
		return nil, errors.New("jwt: Issuer must not be empty")
	}
	return &JWTUtil{cfg: cfg}, nil
}

func (j *JWTUtil) IssueTokenPair(userID uuid.UUID, role string) (access, refresh, hash string, err error) {
	access, err = j.GenerateAccessToken(userID, role)
	if err != nil {
		return "", "", "", fmt.Errorf("jwt: generate access token: %w", err)
	}

	refresh, hash, err = j.GenerateRefreshToken()
	if err != nil {
		return "", "", "", fmt.Errorf("jwt: generate refresh token: %w", err)
	}

	return access, refresh, hash, nil
}

func (j *JWTUtil) GenerateAccessToken(userID uuid.UUID, role string) (string, error) {
	now := time.Now().UTC()
	claims := AccessTokenClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.cfg.Issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.cfg.AccessTokenTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(j.cfg.AccessSecret))
	if err != nil {
		return "", fmt.Errorf("jwt: failed to sign access token: %w", err)
	}
	return signed, nil
}

func (j *JWTUtil) GenerateRefreshToken() (rawToken, tokenHash string, err error) {
	b := make([]byte, 64)
	if _, err = rand.Read(b); err != nil {
		return "", "", fmt.Errorf("jwt: failed to generate refresh token: %w", err)
	}
	// we store only the hash and send the raw to client with httponly cookie
	rawToken = hex.EncodeToString(b)
	tokenHash = j.HashRefreshToken(rawToken)
	return rawToken, tokenHash, nil
}

func (j *JWTUtil) ValidateAccessToken(tokenStr string) (*AccessTokenClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&AccessTokenClaims{},
		func(t *jwt.Token) (interface{}, error) {
			// reject any token not using HS256.
			if t.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(j.cfg.AccessSecret), nil
		},
		jwt.WithIssuer(j.cfg.Issuer),
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuedAt(),
	)
	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, ErrTokenExpired
		case errors.Is(err, jwt.ErrTokenMalformed):
			return nil, ErrTokenMalformed
		default:
			return nil, ErrTokenInvalid
		}
	}

	claims, ok := token.Claims.(*AccessTokenClaims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

func (j *JWTUtil) HashRefreshToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (j *JWTUtil) RefreshTokenTTL() time.Duration {
	return j.cfg.RefreshTokenTTL
}

func (j *JWTUtil) AccessTokenTTL() time.Duration {
	return j.cfg.AccessTokenTTL
}
