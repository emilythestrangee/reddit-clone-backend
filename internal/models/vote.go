package models

import "time"

// Vote model - tracks individual user votes on posts
type Vote struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	UserID    int       `json:"user_id"`
	PostID    int       `json:"post_id"`    // non-zero for post votes
	CommentID int       `json:"comment_id"` // non-zero for comment votes
	VoteType  int       `json:"vote_type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
