package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/emilythestrangee/reddit-clone/backend/internal/models"
)

type CommentHandler struct {
	db *gorm.DB
}

func NewCommentHandler(db *gorm.DB) *CommentHandler {
	return &CommentHandler{db: db}
}

func extractUserID(c *gin.Context) (int, bool) {
	raw, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	switch v := raw.(type) {
	case int:
		return v, true
	case uint:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

func (h *CommentHandler) calculateCommentVotes(commentID int) (int, int) {
	var up, down int64
	h.db.Model(&models.Vote{}).Where("comment_id = ? AND vote_type = ?", commentID, 1).Count(&up)
	h.db.Model(&models.Vote{}).Where("comment_id = ? AND vote_type = ?", commentID, -1).Count(&down)
	return int(up), int(down)
}

// GetComments returns all comments for a post with calculated votes
func (h *CommentHandler) GetComments(c *gin.Context) {
	postID := c.Param("id")
	var comments []models.Comment

	if err := h.db.Where("post_id = ?", postID).Preload("User").Order("created_at desc").Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comments"})
		return
	}

	var responses []gin.H
	for _, comment := range comments {
		up, down := h.calculateCommentVotes(comment.ID)
		responses = append(responses, gin.H{
			"id":         comment.ID,
			"body":       comment.Body,
			"author_id":  comment.AuthorID,
			"post_id":    comment.PostID,
			"user":       comment.User,
			"upvotes":    up,
			"downvotes":  down,
			"created_at": comment.CreatedAt,
			"updated_at": comment.UpdatedAt,
		})
	}

	if responses == nil {
		responses = []gin.H{}
	}

	c.JSON(http.StatusOK, responses)
}

// CreateComment creates a new comment on a post
func (h *CommentHandler) CreateComment(c *gin.Context) {
	var input struct {
		Body string `json:"body" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	postID := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Convert userID safely
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

	// Verify post exists
	var post models.Post
	if err := h.db.First(&post, postID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	comment := models.Comment{
		Body:     input.Body,
		PostID:   post.ID,
		AuthorID: authorID,
	}

	if err := h.db.Create(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
		return
	}

	h.db.Preload("User").First(&comment, comment.ID)
	c.JSON(http.StatusCreated, comment)
}

// UpdateComment updates a comment (owner only)
func (h *CommentHandler) UpdateComment(c *gin.Context) {
	commentID := c.Param("commentId")

	authorID, ok := extractUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var input struct {
		Body string `json:"body" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var comment models.Comment
	if err := h.db.First(&comment, commentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}

	if comment.AuthorID != authorID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only edit your own comments"})
		return
	}

	comment.Body = input.Body
	h.db.Save(&comment)
	h.db.Preload("User").First(&comment, comment.ID)

	up, down := h.calculateCommentVotes(comment.ID)
	c.JSON(http.StatusOK, gin.H{
		"id":         comment.ID,
		"body":       comment.Body,
		"author_id":  comment.AuthorID,
		"post_id":    comment.PostID,
		"user":       comment.User,
		"upvotes":    up,
		"downvotes":  down,
		"created_at": comment.CreatedAt,
		"updated_at": comment.UpdatedAt,
	})
}

// DeleteComment deletes a comment and its votes (owner only)
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	commentID := c.Param("commentId")

	authorID, ok := extractUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var comment models.Comment
	if err := h.db.First(&comment, commentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}

	if comment.AuthorID != authorID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own comments"})
		return
	}

	// Clean up votes on this comment too
	h.db.Where("comment_id = ?", comment.ID).Delete(&models.Vote{})

	if err := h.db.Delete(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete comment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comment deleted successfully"})
}

// UpvoteComment — one vote per user, toggles off if same, switches if opposite
func (h *CommentHandler) UpvoteComment(c *gin.Context) {
	commentID := c.Param("commentId")

	voterID, ok := extractUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var comment models.Comment
	if err := h.db.First(&comment, commentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}

	var existing models.Vote
	err := h.db.Where("user_id = ? AND comment_id = ?", voterID, commentID).First(&existing).Error

	if err == nil {
		if existing.VoteType == 1 {
			// Already upvoted — toggle off
			h.db.Delete(&existing)
			c.JSON(http.StatusOK, gin.H{"message": "Vote removed"})
			return
		}
		// Was a downvote — switch to upvote
		existing.VoteType = 1
		h.db.Save(&existing)
		c.JSON(http.StatusOK, gin.H{"message": "Vote updated"})
		return
	}

	// No vote yet — create upvote
	vote := models.Vote{UserID: voterID, CommentID: comment.ID, VoteType: 1}
	h.db.Create(&vote)
	c.JSON(http.StatusOK, gin.H{"message": "Vote recorded"})
}

// DownvoteComment — one vote per user, toggles off if same, switches if opposite
func (h *CommentHandler) DownvoteComment(c *gin.Context) {
	commentID := c.Param("commentId")

	voterID, ok := extractUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var comment models.Comment
	if err := h.db.First(&comment, commentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}

	var existing models.Vote
	err := h.db.Where("user_id = ? AND comment_id = ?", voterID, commentID).First(&existing).Error

	if err == nil {
		if existing.VoteType == -1 {
			// Already downvoted — toggle off
			h.db.Delete(&existing)
			c.JSON(http.StatusOK, gin.H{"message": "Vote removed"})
			return
		}
		// Was an upvote — switch to downvote
		existing.VoteType = -1
		h.db.Save(&existing)
		c.JSON(http.StatusOK, gin.H{"message": "Vote updated"})
		return
	}

	// No vote yet — create downvote
	vote := models.Vote{UserID: voterID, CommentID: comment.ID, VoteType: -1}
	h.db.Create(&vote)
	c.JSON(http.StatusOK, gin.H{"message": "Vote recorded"})
}
