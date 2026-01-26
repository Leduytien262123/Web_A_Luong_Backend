package router

import (
	"backend/app"
	"backend/internal/handle"
	"backend/internal/repo"
	"backend/utils"

	"github.com/gin-gonic/gin"
)

// SetupAdminRoutes - Routes dành cho super admin và admin
func SetupAdminRoutes(router *gin.Engine) {
	// Khởi tạo repositories và handlers
	userRepo := repo.NewUserRepository(app.GetDB())
	adminHandler := handle.NewAdminHandler(userRepo)
	categoryHandler := handle.NewCategoryHandler()
	articleHandler := handle.NewArticleHandler()
	tagHandler := handle.NewTagHandler()
	s3Handler := handle.NewS3Handler()
	homepageSectionHandler := handle.NewHomepageSectionHandler()

	// Base admin group - yêu cầu authentication
	admin := router.Group("/api/admin")
	admin.Use(utils.AuthMiddleware())

	// Routes chỉ dành cho Super Admin
	superAdminRoutes := admin.Group("/super")
	superAdminRoutes.Use(utils.SuperAdminMiddleware())
	{
		// Quản lý admin users (chỉ super admin)
		superAdminRoutes.GET("/users", adminHandler.GetAllUsers)
		superAdminRoutes.POST("/user", adminHandler.CreateUser)
		superAdminRoutes.GET("/user/:id", adminHandler.GetUserByID)
		superAdminRoutes.PUT("/user/:id", adminHandler.UpdateUser)
		superAdminRoutes.PUT("/user/:id/role", adminHandler.AssignUserRole)
		superAdminRoutes.PUT("/user/:id/status", adminHandler.ToggleUserStatus)
		superAdminRoutes.DELETE("/user/:id", adminHandler.DeleteUser)
		superAdminRoutes.GET("/stats/users", adminHandler.GetUserStats)
	}

	// Routes dành cho cả Super Admin và Admin
	managerRoutes := admin.Group("/manage")
	managerRoutes.Use(utils.AdminMiddleware())
	{
		// Quản lý Danh mục (Categories)
		managerRoutes.GET("/categories", categoryHandler.GetCategories)
		managerRoutes.GET("/category/tree", categoryHandler.GetCategoryTree)
		managerRoutes.GET("/category/:id", categoryHandler.GetCategoryByID)
		managerRoutes.POST("/category", categoryHandler.CreateCategory)
		managerRoutes.PUT("/category/:id", categoryHandler.UpdateCategory)
		managerRoutes.DELETE("/category/:id", categoryHandler.DeleteCategory)

		// Quản lý Article (Bài viết)
		managerRoutes.GET("/articles", articleHandler.GetArticles)
		managerRoutes.GET("/article/:id", articleHandler.GetArticleByID)
		managerRoutes.GET("/article/slug/:slug", articleHandler.GetArticleBySlug)
		managerRoutes.POST("/article", articleHandler.CreateArticle)
		managerRoutes.PUT("/article/:id", articleHandler.UpdateArticle)
		managerRoutes.DELETE("/article/:id", articleHandler.DeleteArticle)

		// Quản lý tags
		managerRoutes.GET("/tags", tagHandler.GetTags)
		managerRoutes.GET("/tags/popular", tagHandler.GetPopularTags)
		managerRoutes.GET("/tag/search", tagHandler.SearchTags)
		managerRoutes.GET("/tag/slug/:slug", tagHandler.GetTagBySlug)
		managerRoutes.GET("/tag/:id", tagHandler.GetTagByID)
		managerRoutes.POST("/tag", tagHandler.CreateTag)
		managerRoutes.PUT("/tag/:id", tagHandler.UpdateTag)
		managerRoutes.DELETE("/tag/:id", tagHandler.DeleteTag)

		// Tìm kiếm và lọc bài viết
		managerRoutes.GET("/articles/search", articleHandler.SearchArticles)
		managerRoutes.GET("/articles/featured", articleHandler.GetFeaturedArticles)

		managerRoutes.POST("/upload/s3", s3Handler.GetUploadUrl)
		managerRoutes.DELETE("/upload", s3Handler.DeleteS3Object)

		// Quản lý Homepage Sections
		managerRoutes.GET("/homepage-sections", homepageSectionHandler.GetSections)
		managerRoutes.GET("/homepage-section/:id", homepageSectionHandler.GetSectionByID)
		managerRoutes.GET("/homepage-section/type/:type_key", homepageSectionHandler.GetSectionByTypeKey)
		managerRoutes.POST("/homepage-section", homepageSectionHandler.CreateSection)
		managerRoutes.PUT("/homepage-section/:id", homepageSectionHandler.UpdateSection)
		managerRoutes.DELETE("/homepage-section/:id", homepageSectionHandler.DeleteSection)
	}
}
