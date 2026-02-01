package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/emilythestrangee/reddit-clone/backend/internal/database"
	"github.com/emilythestrangee/reddit-clone/backend/internal/handlers"
	"github.com/emilythestrangee/reddit-clone/backend/internal/middleware"
)

type Server struct {
	db      *database.Database
	handler *handlers.Handler
}

// NewServer creates and configures a new server
func NewServer() *http.Server {
	// Initialize database
	db, err := database.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Create unified handler
	handler := handlers.NewHandler(db)

	// Create server instance
	newServer := &Server{
		db:      db,
		handler: handler,
	}

	// Configure Gin router
	router := newServer.RegisterRoutes()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // local dev fallback
	}

	// Create HTTP server
	server := &http.Server{
		Addr:         "0.0.0.0:" + port,
		Handler:      router,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Printf("🚀 Server starting on port %s\n", port)
	fmt.Println("📝 Press Ctrl+C to stop the server")

	return server
}

// RegisterRoutes sets up all application routes
func (s *Server) RegisterRoutes() *gin.Engine {
	r := gin.Default()

	// CORS configuration
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * 3600,
	}))

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API routes
	api := r.Group("/api")
	{
		// Auth routes (public)
		api.POST("/register", s.handler.Auth.Register)
		api.POST("/login", s.handler.Auth.Login)

		// OAuth routes
		api.POST("/auth/google", s.handler.Auth.GoogleLogin)
		api.POST("/auth/apple", s.handler.Auth.AppleLogin)

		// Post routes (public reads)
		api.GET("/posts", s.handler.Post.GetPosts)
		api.GET("/posts/:id", s.handler.Post.GetPost)

		// Comment routes (public reads)
		api.GET("/posts/:id/comments", s.handler.Comment.GetComments)

		// User routes (public reads)
		api.GET("/users/:id", s.handler.User.GetUserProfile)
		api.GET("/users/:id/followers", s.handler.User.GetFollowers)
		api.GET("/users/:id/following", s.handler.User.GetFollowing)

		// Protected routes (authentication required)
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			// Auth protected routes
			protected.GET("/me", s.handler.Auth.GetMe)

			// Post protected routes
			protected.POST("/posts", s.handler.Post.CreatePost)
			protected.PUT("/posts/:id", s.handler.Post.UpdatePost)
			protected.DELETE("/posts/:id", s.handler.Post.DeletePost)
			protected.POST("/posts/:id/vote", s.handler.Post.VotePost)

			// Comment protected routes
			protected.POST("/posts/:id/comments", s.handler.Comment.CreateComment)
			protected.POST("/comments/:commentId/upvote", s.handler.Comment.UpvoteComment)
			protected.POST("/comments/:commentId/downvote", s.handler.Comment.DownvoteComment)
			protected.PUT("/comments/:commentId", s.handler.Comment.UpdateComment)
			protected.DELETE("/comments/:commentId", s.handler.Comment.DeleteComment)

			// User protected routes
			protected.PUT("/users/:id", s.handler.User.UpdateUserProfile)
			protected.POST("/users/:id/follow", s.handler.User.FollowUser)
			protected.DELETE("/users/:id/follow", s.handler.User.UnfollowUser)
		}
	}

	return r
}
