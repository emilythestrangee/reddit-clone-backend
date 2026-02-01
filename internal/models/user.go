package models

import "time"

type User struct {
	ID       int    `gorm:"primaryKey" json:"id"`
	Username string `gorm:"unique;not null" json:"username"`
	Email    string `gorm:"unique;not null" json:"email"`
	Password string `gorm:"not null" json:"-"` // For email/password auth
	Bio      string `json:"bio"`
	Avatar   string `json:"avatar"` // Stores avatar ID (1-6) or URL

	// OAuth fields
	GoogleID     string `gorm:"index" json:"-"` // Google user ID
	AppleID      string `gorm:"index" json:"-"` // Apple user ID
	AuthProvider string `json:"auth_provider"`  // "email", "google", "apple"

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Avatar   string `json:"avatar"` // Optional avatar selection
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type OAuthRequest struct {
	Token    string `json:"token" binding:"required"`    // OAuth token from frontend
	Provider string `json:"provider" binding:"required"` // "google" or "apple"
	Username string `json:"username"`                    // Optional, for first-time setup
	Avatar   string `json:"avatar"`                      // Optional, avatar selection
}

type AuthResponse struct {
	Token   string `json:"token"`
	User    User   `json:"user"`
	Message string `json:"message"`
}
