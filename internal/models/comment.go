package models

import "time"

type Comment struct {
	ID              int       `gorm:"primaryKey" json:"id"`
	Body            string    `gorm:"not null" json:"body"`
	AuthorID        int       `json:"author_id"`
	Author          string    `json:"author"`
	User            User      `gorm:"foreignKey:AuthorID" json:"user"`
	PostID          int       `json:"post_id"`
	ParentCommentID *int      `json:"parent_comment_id,omitempty"`
	Upvotes         int       `json:"upvotes"`
	Downvotes       int       `json:"downvotes"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CreateCommentRequest struct {
	Body            string `json:"body"`
	PostID          int    `json:"post_id"`
	ParentCommentID *int   `json:"parent_comment_id,omitempty"`
}
