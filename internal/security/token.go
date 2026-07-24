package security

import (
	"fmt"
	"time"
	"os"
	"errors"
	"context"
	"github.com/golang-jwt/jwt/v5"
)

type TokenService interface {
	GenerateAccessToken(userID, role string) (string, error) 			// 15 min expiry
	GenerateRefreshToken(userID string) (string, error)					// 7 day, single use
	VerifyAccessToken(token string) (*Claims, error)
	RotateRefreshToken(ctx context.Context, oldToken string) (*TokenPair, error)
	RevokeAllForUser(ctx context.Context, userID string) error        	// on logout/password change
}

type Claims struct {
	UserID string
	Role   string
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken string
	RefreshToken string
}

func GenerateAccessToken(userID, role string) (string, error) {
	secret := os.Getenv("JWT_ACCESS_SECRET")

	if secret == "" {
		return "", fmt.Errorf("JWT access secret key is not set in environment variables")
	}

	secretKey := []byte(secret)

	// Create claims payload with expiration metadata
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)), // Expire in 15m
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Create token with HS256 signing algorithm
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret key string
	signedAccessToken, err := token.SignedString(secretKey)

	if err != nil {
		return "", fmt.Errorf("failed to create access token: %w", err)
	}

	return signedAccessToken, nil
}

func GenerateRefreshToken(userID string) (string, error) {
	secret := os.Getenv("JWT_ACCESS_SECRET")

	if secret == "" {
		return "", fmt.Errorf("JWT refresh secret key is not set in environment variables")
	}

	secretKey := []byte(secret)

	// Create claims payload with expiration metadata
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // Expire in 7 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Create token with HS256 signing algorithm
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret key string
	signedRefreshToken, err := token.SignedString(secretKey)

	if err != nil {
		return "", fmt.Errorf("failed to create refresh token: %w", err)
	}

	return signedRefreshToken, nil
}

func VerifyAccessToken(tokenString string) (*Claims, error) {
	secret := os.Getenv("JWT_ACCESS_SECRET")

	if secret == "" {
		return nil, fmt.Errorf("JWT access secret key is not set in environment variables")
	}

	secretKey := []byte(secret)

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		// Verify standard signing algorithm layout
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	// Extract validation context details from metadata fields
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token payload")
}

func RotateRefreshToken(ctx context.Context, oldToken string) (*TokenPair, error) {
	// Verify the old refresh token
	claims, err := VerifyAccessToken(oldToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify old refresh token: %w", err)
	}

	// Generate a new refresh token
	newRefreshToken, err := GenerateRefreshToken(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new refresh token: %w", err)
	}

	// Generate a new access token
	newAccessToken, err := GenerateAccessToken(claims.UserID, claims.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new access token: %w", err)
	}

	return &TokenPair{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func RevokeAllForUser(ctx context.Context, userID string) error {
	// Implement logic to revoke all tokens for the user, e.g., by storing a revocation list in a database or cache.
	// This is a placeholder implementation and should be replaced with actual revocation logic.
	return nil
}