package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/emilythestrangee/reddit-clone/backend/internal/models"
)

type UserHandler struct {
	db *gorm.DB
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{db: db}
}

// GetUserProfile returns a user's profile
func (h *UserHandler) GetUserProfile(c *gin.Context) {
	userID := c.Param("id")
	var user models.User

	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Get user's posts
	var posts []models.Post
	h.db.Where("user_id = ?", userID).Preload("User").Order("created_at desc").Find(&posts)

	// Get follower/following counts
	var followerCount, followingCount int64
	h.db.Model(&models.Follow{}).Where("following_id = ?", userID).Count(&followerCount)
	h.db.Model(&models.Follow{}).Where("follower_id = ?", userID).Count(&followingCount)

	// Check if current user follows this user
	isFollowing := false
	if currentUserID, exists := c.Get("user_id"); exists {
		var follow models.Follow
		err := h.db.Where("follower_id = ? AND following_id = ?", currentUserID, userID).First(&follow).Error
		isFollowing = err == nil
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"bio":      user.Bio,
			"avatar":   user.Avatar,
		},
		"posts":           posts,
		"follower_count":  followerCount,
		"following_count": followingCount,
		"is_following":    isFollowing,
	})
}

func (h *UserHandler) UpdateUserProfile(c *gin.Context) {
	userID := c.Param("id")

	// Get authenticated user ID from middleware
	authUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Check if user is updating their own profile
	if fmt.Sprintf("%v", authUserID) != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only update your own profile"})
		return
	}

	var input struct {
		Bio    string `json:"bio"`
		Avatar string `json:"avatar"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update fields
	if input.Bio != "" {
		user.Bio = input.Bio
	}
	if input.Avatar != "" {
		user.Avatar = input.Avatar
	}

	// Save to database
	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"bio":      user.Bio,
		"avatar":   user.Avatar,
	})
}

// FollowUser follows a user
func (h *UserHandler) FollowUser(c *gin.Context) {
	followingID := c.Param("id")
	followerID, _ := c.Get("user_id")

	// Can't follow yourself
	var followingUser models.User
	if err := h.db.First(&followingUser, followingID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if followingUser.ID == followerID.(int) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot follow yourself"})
		return
	}

	// Check if already following
	var existingFollow models.Follow
	err := h.db.Where("follower_id = ? AND following_id = ?", followerID, followingID).First(&existingFollow).Error
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Already following this user"})
		return
	}

	follow := models.Follow{
		FollowerID:  followerID.(int),
		FollowingID: followingUser.ID,
	}

	if err := h.db.Create(&follow).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to follow user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully followed user"})
}

// UnfollowUser unfollows a user
func (h *UserHandler) UnfollowUser(c *gin.Context) {
	followingID := c.Param("id")
	followerID, _ := c.Get("user_id")

	if err := h.db.Where("follower_id = ? AND following_id = ?", followerID, followingID).Delete(&models.Follow{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unfollow"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully unfollowed user"})
}

// GetFollowers returns a user's followers
func (h *UserHandler) GetFollowers(c *gin.Context) {
	userID := c.Param("id")
	var follows []models.Follow

	h.db.Where("following_id = ?", userID).Preload("Follower").Find(&follows)

	var followers []gin.H
	for _, follow := range follows {
		followers = append(followers, gin.H{
			"id":       follow.Follower.ID,
			"username": follow.Follower.Username,
			"avatar":   follow.Follower.Avatar,
		})
	}

	c.JSON(http.StatusOK, followers)
}

// GetFollowing returns users that a user is following
func (h *UserHandler) GetFollowing(c *gin.Context) {
	userID := c.Param("id")
	var follows []models.Follow

	h.db.Where("follower_id = ?", userID).Preload("Following").Find(&follows)

	var following []gin.H
	for _, follow := range follows {
		following = append(following, gin.H{
			"id":       follow.Following.ID,
			"username": follow.Following.Username,
			"avatar":   follow.Following.Avatar,
		})
	}

	c.JSON(http.StatusOK, following)
}
