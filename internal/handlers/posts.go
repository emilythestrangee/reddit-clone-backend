package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/emilythestrangee/reddit-clone/backend/internal/models"
)

type PostHandler struct {
	db *gorm.DB
}

func NewPostHandler(db *gorm.DB) *PostHandler {
	return &PostHandler{db: db}
}

func (h *PostHandler) calculateVotes(postID int) (int, int) {
	var upvotes, downvotes int64
	h.db.Model(&models.Vote{}).Where("post_id = ? AND vote_type = ?", postID, 1).Count(&upvotes)
	h.db.Model(&models.Vote{}).Where("post_id = ? AND vote_type = ?", postID, -1).Count(&downvotes)
	return int(upvotes), int(downvotes)
}

func (h *PostHandler) GetPosts(c *gin.Context) {
	var posts []models.Post

	if err := h.db.Preload("User").Order("created_at desc").Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch posts"})
		return
	}

	// DON'T embed models.Post — build each response manually
	var responses []gin.H
	for _, post := range posts {
		up, down := h.calculateVotes(post.ID)
		responses = append(responses, gin.H{
			"id":         post.ID,
			"title":      post.Title,
			"body":       post.Body,
			"content":    post.Content,
			"image":      post.Image,
			"user_id":    post.UserID,
			"author_id":  post.AuthorID,
			"community":  post.Community,
			"user":       post.User,
			"upvotes":    up,
			"downvotes":  down,
			"comments":   post.Comments,
			"created_at": post.CreatedAt,
			"updated_at": post.UpdatedAt,
		})
	}

	// If no posts, return empty array not null
	if responses == nil {
		responses = []gin.H{}
	}

	c.JSON(http.StatusOK, responses)
}

// GetPost returns a single post by ID
func (h *PostHandler) GetPost(c *gin.Context) {
	postID := c.Param("id")
	var post models.Post

	if err := h.db.Preload("User").First(&post, postID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	up, down := h.calculateVotes(post.ID)

	c.JSON(http.StatusOK, gin.H{
		"id":         post.ID,
		"title":      post.Title,
		"body":       post.Body,
		"content":    post.Content,
		"image":      post.Image,
		"user_id":    post.UserID,
		"author_id":  post.AuthorID,
		"user":       post.User,
		"upvotes":    up,
		"downvotes":  down,
		"created_at": post.CreatedAt,
		"updated_at": post.UpdatedAt,
	})
}

// CreatePost creates a new post (PROTECTED - requires authentication)
func (h *PostHandler) CreatePost(c *gin.Context) {
	var input struct {
		Title   string `json:"title" binding:"required"`
		Body    string `json:"body"`
		Content string `json:"content"`
		Image   string `json:"image"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Title is required"})
		return
	}

	// Get user ID from auth middleware
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Convert userID to int
	var authorID int
	switch v := userID.(type) {
	case int:
		authorID = v
	case uint:
		authorID = int(v)
	case float64:
		authorID = int(v)
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	// Use content or body (they're the same)
	postContent := input.Content
	if postContent == "" {
		postContent = input.Body
	}

	post := models.Post{
		Title:    input.Title,
		Body:     postContent,
		Content:  postContent,
		Image:    input.Image,
		AuthorID: authorID,
		UserID:   authorID,
	}

	if err := h.db.Create(&post).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create post"})
		return
	}

	// Reload with user information
	h.db.Preload("User").First(&post, post.ID)

	c.JSON(http.StatusCreated, post)
}

// UpdatePost updates an existing post (PROTECTED - requires ownership)
func (h *PostHandler) UpdatePost(c *gin.Context) {
	postID := c.Param("id")

	// Get user ID from auth middleware
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var input struct {
		Title   string `json:"title"`
		Body    string `json:"body"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find the post
	var post models.Post
	if err := h.db.First(&post, postID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	// Convert userID to int for comparison
	var currentUserID int
	switch v := userID.(type) {
	case int:
		currentUserID = v
	case uint:
		currentUserID = int(v)
	case float64:
		currentUserID = int(v)
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	// Check ownership
	if post.AuthorID != currentUserID && post.UserID != currentUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only edit your own posts"})
		return
	}

	// Update fields
	if input.Title != "" {
		post.Title = input.Title
	}
	if input.Body != "" {
		post.Body = input.Body
		post.Content = input.Body
	}
	if input.Content != "" {
		post.Content = input.Content
		post.Body = input.Content
	}

	h.db.Save(&post)
	h.db.Preload("User").First(&post, post.ID)

	c.JSON(http.StatusOK, post)
}

// DeletePost deletes a post (PROTECTED - requires ownership)
func (h *PostHandler) DeletePost(c *gin.Context) {
	postID := c.Param("id")

	// Get user ID from auth middleware
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Find the post
	var post models.Post
	if err := h.db.First(&post, postID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	// Convert userID to int for comparison
	var currentUserID int
	switch v := userID.(type) {
	case int:
		currentUserID = v
	case uint:
		currentUserID = int(v)
	case float64:
		currentUserID = int(v)
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	// Check ownership
	if post.AuthorID != currentUserID && post.UserID != currentUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own posts"})
		return
	}

	if err := h.db.Delete(&post).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete post"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Post deleted successfully"})
}

// VotePost handles upvoting/downvoting a post (PROTECTED - requires authentication)
func (h *PostHandler) VotePost(c *gin.Context) {
	postID := c.Param("id")

	// Get user ID from auth middleware
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var input struct {
		VoteType int `json:"vote_type" binding:"required,oneof=-1 1"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vote type must be -1 or 1"})
		return
	}

	// Check if post exists
	var post models.Post
	if err := h.db.First(&post, postID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	// Convert userID to int
	var voterID int
	switch v := userID.(type) {
	case int:
		voterID = v
	case uint:
		voterID = int(v)
	case float64:
		voterID = int(v)
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	// Check if user already voted
	var existingVote models.Vote
	err := h.db.Where("user_id = ? AND post_id = ?", voterID, postID).First(&existingVote).Error

	if err == nil {
		// User already voted
		if existingVote.VoteType == input.VoteType {
			// Same vote - remove it (toggle)
			h.db.Delete(&existingVote)
			c.JSON(http.StatusOK, gin.H{"message": "Vote removed"})
			return
		} else {
			// Different vote - update it
			existingVote.VoteType = input.VoteType
			h.db.Save(&existingVote)
			c.JSON(http.StatusOK, gin.H{"message": "Vote updated"})
			return
		}
	}

	// Create new vote
	vote := models.Vote{
		UserID:   voterID,
		PostID:   post.ID,
		VoteType: input.VoteType,
	}

	if err := h.db.Create(&vote).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to vote"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Vote recorded"})
}

// GetUserPosts returns all posts by a specific user
func (h *PostHandler) GetUserPosts(c *gin.Context) {
	userID := c.Param("id")
	var posts []models.Post

	if err := h.db.Preload("User").Where("user_id = ? OR author_id = ?", userID, userID).Order("created_at desc").Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user posts"})
		return
	}

	c.JSON(http.StatusOK, posts)
}
