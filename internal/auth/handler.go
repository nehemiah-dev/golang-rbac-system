package auth

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	authService Service
}

func NewHandler(authService Service) *Handler {
	return &Handler{
		authService: authService,
	}
}

// RegisterUser godoc
// @Summary Register a new user
// @Description Creates a new user account with email and password
// @Tags users
// @Accept json
// @Produce json
// @Param user body RegisterWithPasswordRequest true "User registration payload"
// @Success 201 {object} RegisterUserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/register [post]
func (h *Handler) RegisterUser(c *gin.Context) {
	var input RegisterWithPasswordRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		return
	}

	user, err := h.authService.RegisterWithPassword(c, input.Email, input.Password)
	if err != nil {
		log.Println("Error occurred while trying to register user:", err)

		if errors.Is(err, ErrUserWithEmailAlreadyExists) {
			c.JSON(http.StatusConflict, ErrorResponse{Message: err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, RegisterUserResponse{
		Message: "User registered successfully",
		Data: UserResponse{
			ID:        user.ID.String(),
			Email:     user.Email,
			Role:      user.Role,
			Verified:  user.IsVerified,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
		},
	})
}

// LoginUser godoc
// @Summary Log in with email and password
// @Description Logs in a user account with email and password
// @Tags users
// @Accept json
// @Produce json
// @Param user body LoginWithPasswordRequest true "User login payload"
// @Success 200 {object} LoginUserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/login [post]
func (h *Handler) LoginUser(c *gin.Context) {
	var input LoginWithPasswordRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		return
	}

	pair, err := h.authService.LoginWithPassword(c, input.Email, input.Password)
	if err != nil {
		log.Println("Error occurred while trying to log in user:", err)

		if errors.Is(err, ErrNonExistentUser) || errors.Is(err, ErrPasswordMismatchDuringLogin) {
			c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid credentials. Please try again."})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal server error"})
		return
	}

	// Setting refresh token in httponly cookie
	c.SetCookie("refresh_token", pair.RefreshToken, 604800, "/", "", false, true)

	c.JSON(http.StatusOK, LoginUserResponse{
		Message: "User logged in successfully",
		Data: LoginData{
			AccessToken: pair.AccessToken,
			ExpiresIn:   pair.ExpiresIn,
			TokenType:   "Bearer",
			User: UserResponse{
				ID:        pair.User.ID.String(),
				Email:     pair.User.Email,
				Role:      pair.User.Role,
				Verified:  pair.User.IsVerified,
				CreatedAt: pair.User.CreatedAt.Format(time.RFC3339),
			},
		},
	})
}

// RefreshTokens godoc
// @Summary Refresh access token
// @Description Issues a new access/refresh token pair using a valid refresh token
// @Tags users
// @Accept json
// @Produce json
// @Param user body RefreshTokenRequest true "Refresh token payload"
// @Success 200 {object} RefreshTokenResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/refresh [post]
func (h *Handler) RefreshTokens(c *gin.Context) {
	var input RefreshTokenRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		return
	}

	pair, err := h.authService.RefreshTokens(c, input.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, ErrRefreshTokenInvalid),
			errors.Is(err, ErrRefreshTokenExpired),
			errors.Is(err, ErrRefreshTokenReuse):

			c.JSON(http.StatusUnauthorized, ErrorResponse{Message: err.Error()})

		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal server error"})
		}
		return
	}

	c.SetCookie("refresh_token", pair.RefreshToken, 604800, "/", "", false, true)

	c.JSON(http.StatusOK, RefreshTokenResponse{
		Message: "Tokens refreshed successfully",
		Data: LoginData{
			AccessToken: pair.AccessToken,
			ExpiresIn:   pair.ExpiresIn,
			TokenType:   "Bearer",
			User: UserResponse{
				ID:        pair.User.ID.String(),
				Email:     pair.User.Email,
				Role:      pair.User.Role,
				Verified:  pair.User.IsVerified,
				CreatedAt: pair.User.CreatedAt.Format(time.RFC3339),
			},
		},
	})
}

// Logout godoc
// @Summary Log out user
// @Description Revokes the given refresh token, ending the session
// @Tags users
// @Accept json
// @Produce json
// @Param user body LogoutRequest true "Logout payload"
// @Success 200 {object} LogoutResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	var input LogoutRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		return
	}

	if err := h.authService.Logout(c, input.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal server error"})
		return
	}

	c.JSON(http.StatusOK, LogoutResponse{Message: "Logged out successfully"})
}
