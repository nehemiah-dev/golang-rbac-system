package security

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Steve-s-Circle-on-System-Design/golang-rbac-system/internal/auth"
	usersdb "github.com/Steve-s-Circle-on-System-Design/golang-rbac-system/internal/users/sqlc"
)

// TokenService issues and validates JWT access tokens and opaque,
// DB-backed refresh tokens. It is a thin orchestration layer over
// auth.JWTUtil (signing/hashing) and the refresh_tokens table
// (single-use + revocation state).
type TokenService interface {
	GenerateAccessToken(userID, role string) (string, error)                 // 15 min expiry
	GenerateRefreshToken(ctx context.Context, userID string) (string, error) // 7 day, single use
	VerifyAccessToken(token string) (*Claims, error)
	RotateRefreshToken(ctx context.Context, oldToken string) (*TokenPair, error)
	RevokeAllForUser(ctx context.Context, userID string) error // on logout/password change
}

type Claims struct {
	UserID string
	Role   string
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type jwtTokenService struct {
	jwtUtil *auth.JWTUtil
	repo    *auth.Repository
}

var _ TokenService = (*jwtTokenService)(nil)

// NewJWTTokenService builds a TokenService backed by the given JWTUtil
// (token signing/hashing) and database pool (refresh_tokens store).
func NewJWTTokenService(jwtUtil *auth.JWTUtil, pool *pgxpool.Pool) TokenService {
	return &jwtTokenService{
		jwtUtil: jwtUtil,
		repo:    auth.NewRepository(pool),
	}
}

func (s *jwtTokenService) GenerateAccessToken(userID, role string) (string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("invalid user id: %w", err)
	}

	return s.jwtUtil.GenerateAccessToken(uid, role)
}

func (s *jwtTokenService) GenerateRefreshToken(ctx context.Context, userID string) (string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("invalid user id: %w", err)
	}

	rawToken, tokenHash, err := s.jwtUtil.GenerateRefreshToken()
	if err != nil {
		return "", err
	}

	_, err = s.repo.CreateRefreshToken(ctx, usersdb.CreateRefreshTokenParams{
		UserID:    uid,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().UTC().Add(s.jwtUtil.RefreshTokenTTL()),
	})
	if err != nil {
		return "", fmt.Errorf("failed to persist refresh token: %w", err)
	}

	return rawToken, nil
}

func (s *jwtTokenService) VerifyAccessToken(tokenString string) (*Claims, error) {
	claims, err := s.jwtUtil.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, err
	}

	return &Claims{
		UserID:           claims.UserID.String(),
		Role:             claims.Role,
		RegisteredClaims: claims.RegisteredClaims,
	}, nil
}

// RotateRefreshToken verifies oldToken against the refresh_tokens store,
// rejects it if expired or already used (revoking every other session for
// the user on reuse, since reuse of a single-use token indicates theft),
// and — only once oldToken is confirmed valid and unused — atomically
// revokes it and issues a fresh access/refresh pair.
func (s *jwtTokenService) RotateRefreshToken(ctx context.Context, oldToken string) (*TokenPair, error) {
	hash := s.jwtUtil.HashRefreshToken(oldToken)

	stored, err := s.repo.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, auth.ErrRefreshTokenInvalid
		}
		return nil, err
	}

	if stored.IsRevoked {
		_ = s.repo.RevokeAllRefreshTokensForUser(ctx, stored.UserID)
		return nil, auth.ErrRefreshTokenReuse
	}

	if time.Now().UTC().After(stored.ExpiresAt) {
		return nil, auth.ErrRefreshTokenExpired
	}

	user, err := s.repo.GetByID(ctx, stored.UserID)
	if err != nil {
		return nil, err
	}

	if err := s.repo.RevokeRefreshToken(ctx, stored.ID); err != nil {
		return nil, err
	}

	accessToken, refreshToken, refreshHash, err := s.jwtUtil.IssueTokenPair(user.ID, user.Role)
	if err != nil {
		return nil, err
	}

	_, err = s.repo.CreateRefreshToken(ctx, usersdb.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: refreshHash,
		ExpiresAt: time.Now().UTC().Add(s.jwtUtil.RefreshTokenTTL()),
	})
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *jwtTokenService) RevokeAllForUser(ctx context.Context, userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	return s.repo.RevokeAllRefreshTokensForUser(ctx, uid)
}
