package auth

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	usersdb "github.com/Steve-s-Circle-on-System-Design/golang-rbac-system/internal/users/sqlc"
)

var (
	ErrUserWithEmailAlreadyExists  = errors.New("user with that email already exists")
	ErrNonExistentUser             = errors.New("user doesn't exist with that email")
	ErrPasswordMismatchDuringLogin = errors.New("invalid password")
	ErrRefreshTokenInvalid         = errors.New("refresh token is invalid")
	ErrRefreshTokenExpired         = errors.New("refresh token has expired")
	ErrRefreshTokenReuse           = errors.New("refresh token reuse detected — all sessions have been terminated")
)

type TokenPair struct {
	User         usersdb.User
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64 // Life span of access token
}

type Service interface {
	RegisterWithPassword(ctx context.Context, email, password string) (usersdb.User, error)
	LoginWithPassword(ctx context.Context, email, password string) (*TokenPair, error)
	Logout(ctx context.Context, rawRefreshToken string) error
	RefreshTokens(ctx context.Context, rawRefreshToken string) (*TokenPair, error)
}

type authService struct {
	Repository *Repository
	jwtUtil    *JWTUtil
}

func NewService(repository *Repository, jwtUtil *JWTUtil) Service {
	return &authService{
		Repository: repository,
		jwtUtil:    jwtUtil,
	}
}

func (s *authService) RegisterWithPassword(ctx context.Context, email, password string) (usersdb.User, error) {
	_, err := s.Repository.GetByEmail(ctx, email)
	if err == nil {
		return usersdb.User{}, ErrUserWithEmailAlreadyExists
	} else if !errors.Is(err, pgx.ErrNoRows) {
		log.Println("failed to check existing user:", err)
		return usersdb.User{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Something went wrong while hashing the password", err.Error())
		return usersdb.User{}, err
	}

	user, err := s.Repository.Create(ctx, usersdb.CreateUserParams{
		Email:        email,
		PasswordHash: string(passwordHash),
	})
	if err != nil {
		log.Println("Something went wrong while trying to save the new user in the db", err.Error())
		return usersdb.User{}, err
	}
	return user, nil
}

func (s *authService) LoginWithPassword(ctx context.Context, email, password string) (*TokenPair, error) {
	existingUser, err := s.Repository.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			dummyHash := "$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewrGQmMDL4g0/tKu"
			_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password))
			return nil, ErrNonExistentUser
		}
		log.Println("failed to check existing user:", err)
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(existingUser.PasswordHash), []byte(password))
	if err != nil {
		return nil, ErrPasswordMismatchDuringLogin
	}

	accessToken, refreshToken, refreshHash, err := s.jwtUtil.IssueTokenPair(
		existingUser.ID,
		existingUser.Role,
	)
	if err != nil {
		log.Println("failed to issue token pair:", err)
		return nil, err
	}

	expiresAt := time.Now().UTC().Add(s.jwtUtil.RefreshTokenTTL())

	_, err = s.Repository.CreateRefreshToken(
		ctx,
		usersdb.CreateRefreshTokenParams{
			UserID:    existingUser.ID,
			TokenHash: refreshHash,
			ExpiresAt: expiresAt,
		},
	)
	if err != nil {
		log.Println("failed to save refresh token:", err)
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwtUtil.AccessTokenTTL().Seconds()),
		User:         existingUser,
	}, nil
}

func (s *authService) Logout(ctx context.Context, rawRefreshToken string) error {
	hash := s.jwtUtil.HashRefreshToken(rawRefreshToken)

	token, err := s.Repository.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}

	if token.IsRevoked {
		return nil
	}

	return s.Repository.RevokeRefreshToken(ctx, token.ID)
}

func (s *authService) RefreshTokens(ctx context.Context, rawRefreshToken string) (*TokenPair, error) {
	hash := s.jwtUtil.HashRefreshToken(rawRefreshToken)

	token, err := s.Repository.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRefreshTokenInvalid
		}
		return nil, err
	}

	if token.IsRevoked {
		_ = s.Repository.RevokeAllRefreshTokensForUser(ctx, token.UserID)
		return nil, ErrRefreshTokenReuse
	}

	if time.Now().UTC().After(token.ExpiresAt) {
		return nil, ErrRefreshTokenExpired
	}

	user, err := s.Repository.GetByID(ctx, token.UserID)
	if err != nil {
		return nil, err
	}

	err = s.Repository.RevokeRefreshToken(ctx, token.ID)
	if err != nil {
		return nil, err
	}

	accessToken, refreshToken, refreshHash, err := s.jwtUtil.IssueTokenPair(user.ID, user.Role)
	if err != nil {
		return nil, err
	}

	newExpiry := time.Now().UTC().Add(s.jwtUtil.RefreshTokenTTL())

	_, err = s.Repository.CreateRefreshToken(ctx, usersdb.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: refreshHash,
		ExpiresAt: newExpiry,
	})
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwtUtil.AccessTokenTTL().Seconds()),
		User:         user,
	}, nil
}
