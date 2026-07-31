package auth

import (
	"errors"
	"log"
	"net/http"

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

func (h *Handler) RegisterUser(c *gin.Context) {
	var input RegisterWithPasswordRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.authService.RegisterWithPassword(c, input.Email, input.Password)
	if err != nil {
		log.Println("Error occurred while trying to register user:", err)

		if errors.Is(err, ErrUserWithEmailAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
	})
}

func (h *Handler) LoginUser(c *gin.Context) {
	var input LoginWithPasswordRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pair, err := h.authService.LoginWithPassword(c, input.Email, input.Password)
	if err != nil {
		log.Println("Error occurred while trying to register user:", err)

		if errors.Is(err, ErrNonExistentUser) || errors.Is(err, ErrPasswordMismatchDuringLogin) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid credentials. Please try again."})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	// Setting refresh token in httponly cookie
	c.SetCookie("refresh_token", pair.RefreshToken, 604800, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message":     "User logged in successfully",
		"accessToken": pair.AccessToken,
		"expiresIn":   pair.ExpiresIn,
		"tokenType":   "Bearer",
	})
}

func (h *Handler) RefreshTokens(c *gin.Context) {
	var input RefreshTokenRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	pair, err := h.authService.RefreshTokens(c, input.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, ErrRefreshTokenInvalid),
			errors.Is(err, ErrRefreshTokenExpired),
			errors.Is(err, ErrRefreshTokenReuse):

			c.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}
		return
	}

	c.SetCookie("refresh_token", pair.RefreshToken, 604800, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"accessToken": pair.AccessToken,
		"expiresIn":   pair.ExpiresIn,
		"tokenType":   "Bearer",
	})
}

func (h *Handler) Logout(c *gin.Context) {
	var input LogoutRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := h.authService.Logout(c, input.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}
