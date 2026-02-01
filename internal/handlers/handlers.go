package handlers

import (
	"github.com/emilythestrangee/reddit-clone/backend/internal/database"
)

// Handler combines all handler types
type Handler struct {
	Auth    *AuthHandler
	Post    *PostHandler
	Comment *CommentHandler
	User    *UserHandler
}

// NewHandler creates a unified handler with all sub-handlers
func NewHandler(db *database.Database) *Handler {
	// Get the GORM DB instance from the service
	dbService := database.New()
	gormDB := dbService.GetDB()

	return &Handler{
		Auth:    NewAuthHandler(gormDB),
		Post:    NewPostHandler(gormDB),
		Comment: NewCommentHandler(gormDB),
		User:    NewUserHandler(gormDB),
	}
}
