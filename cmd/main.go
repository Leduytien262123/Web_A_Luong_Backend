package main

import (
	"backend/app"
	"backend/router"
	"backend/utils"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Connect to database and initialize
	app.Connect()

	// Set Gin mode
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize Gin router
	r := gin.New()

	// Add basic middleware
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// DISABLE auto redirect trailing slash
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	// Add CORS middleware FIRST - before any other middleware
	r.Use(utils.CORSMiddleware())

	// Setup routes
	router.SetupAuthRoutes(r)
	router.SetupAdminRoutes(r)
	router.SetupCategoryRoutes(r)
	router.SetupProductRoutes(r)
	router.SetupOrderRoutes(r)
	router.SetupCartRoutes(r)
	router.SetupNewsRoutes(r)

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "OK",
			"message": "Server is running",
		})
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port changed from 3000 to 8080
	}

	log.Printf("Server starting on port %s", port)
	log.Printf("Server will be available at: http://localhost:%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Printf("Failed to start server on port %s: %v", port, err)
		log.Println("Tip: Port might be in use. Try changing PORT in .env file")
		log.Fatal(err)
	}
}