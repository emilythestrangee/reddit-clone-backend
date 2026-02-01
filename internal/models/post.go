package models

import "time"

type Post struct {
	ID          int       `gorm:"primaryKey" json:"id"`
	Title       string    `gorm:"not null" json:"title"`
	Body        string    `json:"body,omitempty"`
	Content     string    `json:"content"`
	Image       string    `json:"image"`
	UserID      int       `json:"user_id"`
	AuthorID    int       `json:"author_id"`
	Author      string    `json:"author"`
	CommunityID int       `json:"community_id"`
	Community   string    `json:"community"`
	Comments    int       `json:"comments"`
	CreatedAt   time.Time `json:"created_at"`
	User        User      `gorm:"foreignKey:UserID" json:"user"`
	Upvotes     int       `gorm:"default:0" json:"upvotes"`
	Downvotes   int       `gorm:"default:0" json:"downvotes"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreatePostRequest struct {
	Title       string `json:"title"`
	Body        string `json:"body"`
	CommunityID int    `json:"community_id"`
}
