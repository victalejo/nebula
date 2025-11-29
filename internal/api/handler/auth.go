package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/victalejo/nebula/internal/core/logger"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	jwtSecret     string
	tokenDuration time.Duration
	log           logger.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(jwtSecret string, tokenDuration time.Duration, log logger.Logger) *AuthHandler {
	return &AuthHandler{
		jwtSecret:     jwtSecret,
		tokenDuration: tokenDuration,
		log:           log,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// TokenResponse represents a token response
type TokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	// TODO: Implement proper user authentication
	// For now, use a simple hardcoded check for development
	if req.Username != "admin" || req.Password != "admin" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid credentials",
		})
		return
	}

	// Generate JWT token
	expiresAt := time.Now().Add(h.tokenDuration)

	claims := jwt.MapClaims{
		"sub":      "user-1",
		"username": req.Username,
		"exp":      expiresAt.Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		h.log.Error("failed to sign token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate token",
		})
		return
	}

	c.JSON(http.StatusOK, TokenResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	})
}

// Refresh handles token refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	// Get current token from header
	tokenString := c.GetHeader("Authorization")
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}

	// Parse existing token (even if expired)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.jwtSecret), nil
	})

	if err != nil && !isExpiredError(err) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid token",
		})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid token claims",
		})
		return
	}

	// Generate new token
	expiresAt := time.Now().Add(h.tokenDuration)

	newClaims := jwt.MapClaims{
		"sub":      claims["sub"],
		"username": claims["username"],
		"exp":      expiresAt.Unix(),
		"iat":      time.Now().Unix(),
	}

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	newTokenString, err := newToken.SignedString([]byte(h.jwtSecret))
	if err != nil {
		h.log.Error("failed to sign token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate token",
		})
		return
	}

	c.JSON(http.StatusOK, TokenResponse{
		Token:     newTokenString,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	})
}

func isExpiredError(err error) bool {
	return err.Error() == "token has invalid claims: token is expired"
}
