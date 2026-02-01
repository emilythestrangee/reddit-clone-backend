package models

import "time"

// Follow model
type Follow struct {
	ID          int       `gorm:"primaryKey" json:"id"`
	FollowerID  int       `json:"follower_id"`
	FollowingID int       `json:"following_id"`
	Follower    User      `gorm:"foreignKey:FollowerID" json:"follower"`
	Following   User      `gorm:"foreignKey:FollowingID" json:"following"`
	CreatedAt   time.Time `json:"created_at"`
}
