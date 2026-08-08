package auth

type RegisterWithPasswordRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=32"`
}

type LoginWithPasswordRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=32"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type SuccessResponse[T any] struct {
	Data    T      `json:"data"`
	Message string `json:"message" example:"User registered successfully"`
}

// ErrorResponse is the standard error envelope returned by all auth endpoints.
type ErrorResponse struct {
	Message string `json:"message" example:"something went wrong"`
}

// UserResponse is the public-facing representation of a user.
type UserResponse struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email     string `json:"email" example:"user@example.com"`
	Role      string `json:"role" example:"USER"`
	CreatedAt string `json:"createdAt" example:"2026-08-05T09:00:00Z"`
	Verified  bool   `json:"isVerified" example:"false"`
}

// RegisterUserResponse is returned on successful registration.
type RegisterUserResponse struct {
	Message string       `json:"message" example:"User registered successfully"`
	Data    UserResponse `json:"data"`
}

// LoginData is the token + user payload shared by login and refresh responses.
type LoginData struct {
	AccessToken string       `json:"accessToken"`
	TokenType   string       `json:"tokenType" example:"Bearer"`
	User        UserResponse `json:"user"`
	ExpiresIn   int64        `json:"expiresIn" example:"900"`
}

// LoginUserResponse is returned on successful login.
type LoginUserResponse struct {
	Message string    `json:"message" example:"User logged in successfully"`
	Data    LoginData `json:"data"`
}

// RefreshTokenResponse is returned on successful token refresh.
type RefreshTokenResponse struct {
	Message string    `json:"message" example:"Tokens refreshed successfully"`
	Data    LoginData `json:"data"`
}

// LogoutResponse is returned on successful logout.
type LogoutResponse struct {
	Message string `json:"message" example:"Logged out successfully"`
}
