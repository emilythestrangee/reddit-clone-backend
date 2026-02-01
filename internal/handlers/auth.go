package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/emilythestrangee/reddit-clone/backend/internal/models"
)

type AuthHandler struct {
	db *gorm.DB
}

func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

// GoogleUserInfo represents user data from Google OAuth
type GoogleUserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Picture       string `json:"picture"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
}

// AppleUserInfo represents user data from Apple Sign In
type AppleUserInfo struct {
	Sub            string `json:"sub"`
	Email          string `json:"email"`
	EmailVerified  string `json:"email_verified"`
	IsPrivateEmail string `json:"is_private_email"`
}

// verifyGoogleIDToken verifies the Google ID token and returns user info
func verifyGoogleIDToken(idToken string) (*GoogleUserInfo, error) {
	resp, err := http.Get(
		"https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid google token")
	}

	var user GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	if !user.EmailVerified {
		return nil, fmt.Errorf("email not verified")
	}

	return &user, nil
}

// verifyAppleIDToken verifies Apple ID token (simplified version)
// In production, you should use Apple's public keys to verify JWT signature
func verifyAppleIDToken(idToken string) (*AppleUserInfo, error) {
	// Split the JWT token
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Decode the payload (base64)
	// Note: In production, you MUST verify the signature using Apple's public keys
	// This is a simplified version for demonstration

	// For now, we'll just parse the token claims
	token, err := jwt.Parse(idToken, func(token *jwt.Token) (interface{}, error) {
		// In production, fetch and use Apple's public keys
		return []byte("dummy-key-for-parsing"), nil
	})

	if err != nil {
		// If parsing fails, return error
		// In production, you should properly verify with Apple's keys
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	user := &AppleUserInfo{
		Sub:   claims["sub"].(string),
		Email: claims["email"].(string),
	}

	return user, nil
}

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
		Avatar   string `json:"avatar"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if username or email already exists
	var existingUser models.User
	if err := h.db.Where("username = ? OR email = ?", input.Username, input.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username or email already exists"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user := models.User{
		Username:     input.Username,
		Email:        input.Email,
		Password:     string(hashedPassword),
		Avatar:       input.Avatar,
		AuthProvider: "email",
	}

	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Generate JWT token AFTER creating user
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
		"exp":      time.Now().Add(time.Hour * 72).Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"token":   tokenString, // ✅ ADD TOKEN
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"avatar":   user.Avatar,
		},
	})
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := h.db.Where("email = ? AND auth_provider = ?", input.Email, "email").First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
		"exp":      time.Now().Add(time.Hour * 72).Unix(), // 72 hours
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   tokenString,
		"user": gin.H{
			"id":            user.ID,
			"username":      user.Username,
			"email":         user.Email,
			"bio":           user.Bio,
			"avatar":        user.Avatar,
			"auth_provider": user.AuthProvider,
		},
	})
}

// GoogleLogin handles Google OAuth login
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	var input struct {
		Token    string `json:"token" binding:"required"`
		Username string `json:"username"`
		Avatar   string `json:"avatar"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify Google ID token
	googleUser, err := verifyGoogleIDToken(input.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Google token"})
		return
	}

	var user models.User
	result := h.db.Where("email = ? OR google_id = ?", googleUser.Email, googleUser.Sub).First(&user)

	if result.Error == gorm.ErrRecordNotFound {
		// Create new user from Google account
		username := input.Username
		if username == "" {
			username = generateUsernameFromEmail(googleUser.Email)
		}

		// Ensure username is unique
		username = h.ensureUniqueUsername(username)

		avatar := input.Avatar
		if avatar == "" && googleUser.Picture != "" {
			avatar = googleUser.Picture
		}

		user = models.User{
			Username:     username,
			Email:        googleUser.Email,
			Avatar:       avatar,
			GoogleID:     googleUser.Sub,
			AuthProvider: "google",
			Password:     "", // No password for OAuth users
		}

		if err := h.db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}
	} else if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	} else {
		// Existing user - update Google ID if not set
		if user.GoogleID == "" {
			user.GoogleID = googleUser.Sub
			h.db.Save(&user)
		}
		// Update avatar if provided and user doesn't have one
		if input.Avatar != "" && user.Avatar == "" {
			user.Avatar = input.Avatar
			h.db.Save(&user)
		}
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
		"exp":      time.Now().Add(72 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
		"user": gin.H{
			"id":            user.ID,
			"username":      user.Username,
			"email":         user.Email,
			"avatar":        user.Avatar,
			"bio":           user.Bio,
			"auth_provider": user.AuthProvider,
		},
	})
}

// AppleLogin handles Apple Sign In
func (h *AuthHandler) AppleLogin(c *gin.Context) {
	var input struct {
		Token    string `json:"token" binding:"required"`
		Username string `json:"username"`
		Avatar   string `json:"avatar"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify Apple ID token
	appleUser, err := verifyAppleIDToken(input.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Apple token"})
		return
	}

	var user models.User
	result := h.db.Where("email = ? OR apple_id = ?", appleUser.Email, appleUser.Sub).First(&user)

	if result.Error == gorm.ErrRecordNotFound {
		// Create new user from Apple account
		username := input.Username
		if username == "" {
			username = generateUsernameFromEmail(appleUser.Email)
		}

		// Ensure username is unique
		username = h.ensureUniqueUsername(username)

		user = models.User{
			Username:     username,
			Email:        appleUser.Email,
			Avatar:       input.Avatar,
			AppleID:      appleUser.Sub,
			AuthProvider: "apple",
			Password:     "", // No password for OAuth users
		}

		if err := h.db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}
	} else if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	} else {
		// Existing user - update Apple ID if not set
		if user.AppleID == "" {
			user.AppleID = appleUser.Sub
			h.db.Save(&user)
		}
		// Update avatar if provided and user doesn't have one
		if input.Avatar != "" && user.Avatar == "" {
			user.Avatar = input.Avatar
			h.db.Save(&user)
		}
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
		"exp":      time.Now().Add(72 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
		"user": gin.H{
			"id":            user.ID,
			"username":      user.Username,
			"email":         user.Email,
			"avatar":        user.Avatar,
			"bio":           user.Bio,
			"auth_provider": user.AuthProvider,
		},
	})
}

// GetMe returns the current authenticated user
func (h *AuthHandler) GetMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            user.ID,
		"username":      user.Username,
		"email":         user.Email,
		"bio":           user.Bio,
		"avatar":        user.Avatar,
		"auth_provider": user.AuthProvider,
		"created_at":    user.CreatedAt,
	})
}

// Helper functions

func generateUsernameFromEmail(email string) string {
	for i, c := range email {
		if c == '@' {
			return email[:i]
		}
	}
	return email
}

func (h *AuthHandler) ensureUniqueUsername(baseUsername string) string {
	username := baseUsername
	counter := 1

	for {
		var existingUser models.User
		if err := h.db.Where("username = ?", username).First(&existingUser).Error; err == gorm.ErrRecordNotFound {
			// Username is available
			return username
		}
		// Username exists, try with counter
		username = fmt.Sprintf("%s%d", baseUsername, counter)
		counter++
	}
}
