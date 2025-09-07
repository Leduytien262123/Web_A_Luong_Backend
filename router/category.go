package router

import (
	"backend/internal/handle"
	"backend/utils"

	"github.com/gin-gonic/gin"
)

func SetupCategoryRoutes(r *gin.Engine) {
	categoryHandler := handle.NewCategoryHandler()

	// Routes công khai
	publicRoutes := r.Group("/api/categories")
	{
		publicRoutes.GET("/", categoryHandler.GetCategories)
		publicRoutes.GET("/:id", categoryHandler.GetCategoryByID)
		publicRoutes.GET("/slug/:slug", categoryHandler.GetCategoryBySlug)
	}

	// Routes được bảo vệ (admin và owner)
	adminRoutes := r.Group("/api/admin/manage")
	adminRoutes.Use(utils.AuthMiddleware())
	adminRoutes.Use(utils.AdminMiddleware()) // Bây giờ cho phép cả admin và owner
	{
		adminRoutes.GET("/categories", categoryHandler.GetCategories) 
		adminRoutes.GET("/categories/:id", categoryHandler.GetCategoryByID)
		adminRoutes.POST("/categories", categoryHandler.CreateCategory)
		adminRoutes.PUT("/categories/:id", categoryHandler.UpdateCategory)
		adminRoutes.DELETE("/categories/:id", categoryHandler.DeleteCategory)
	}
}
