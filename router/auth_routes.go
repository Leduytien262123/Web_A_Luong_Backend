package router

import (
	"backend/app"
	"backend/internal/handle"
	"backend/internal/repo"
	"backend/utils"

	"github.com/gin-gonic/gin"
)

// SetupAuthRoutes - Routes xác thực và các API ngoài lề
func SetupAuthRoutes(router *gin.Engine) {
	// Khởi tạo repository và handler
	userRepo := repo.NewUserRepository(app.GetDB())
	authHandler := handle.NewAuthHandler(userRepo)

	// Routes xác thực công khai
	auth := router.Group("/api/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	// Routes xác thực được bảo vệ
	protected := router.Group("/api/auth")
	protected.Use(utils.AuthMiddleware())
	{
		protected.GET("/profile", authHandler.GetProfile)
		protected.PUT("/profile", authHandler.UpdateProfile)
		protected.PUT("/password", authHandler.ChangePassword)
	}

	// Health check endpoint - API ngoài lề
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "OK",
			"message": "Server is running",
		})
	})

	// API version info - API ngoài lề
	router.GET("/api/version", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version": "1.0.0",
			"name":    "Shop Backend API",
		})
	})

	// Server info - API ngoài lề
	router.GET("/api/info", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"server": "Shop Backend",
			"status": "running",
			"endpoints": gin.H{
				"auth":   "/api/auth/*",
				"public": "/api/*",
				"admin":  "/api/admin/*",
				"health": "/health",
			},
		})
	})
}
