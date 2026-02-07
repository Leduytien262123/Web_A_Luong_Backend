package router

import (
	"backend/internal/handle"

	"github.com/gin-gonic/gin"
)

// SetupUserRoutes - Routes công khai để xem bài viết
func SetupUserRoutes(router *gin.Engine) {
	// Khởi tạo handlers
	categoryHandler := handle.NewCategoryHandler()
	articleHandler := handle.NewArticleHandler()
	tagHandler := handle.NewTagHandler()
	s3Handler := handle.NewS3Handler()
	homepageSectionHandler := handle.NewHomepageSectionHandler()

	// Routes công khai - không cần xác thực
	public := router.Group("/api")
	{
		public.POST("/upload/s3", s3Handler.GetUploadUrl)

		publicCategories := public.Group("/categories")
		{
			publicCategories.GET("", categoryHandler.GetPublicCategories)
			publicCategories.GET("/home", categoryHandler.GetPublicHomeCategories)
			publicCategories.GET("/:slug", articleHandler.GetArticlesByCategorySlug)
		}

		// Article công khai
		publicArticles := public.Group("/articles")
		{
			publicArticles.GET("/featured", articleHandler.GetFeaturedArticles)
			publicArticles.GET("/all", articleHandler.GetAllPublicArticles)
			publicArticles.GET("", articleHandler.GetPublicArticles)
			publicArticles.GET("/:slug", articleHandler.GetArticleBySlugPublic)
		}

		// Tag công khai
		publicTags := public.Group("/tags")
		{
			publicTags.GET("", tagHandler.GetPublicTags)
			publicTags.GET("/:slug", tagHandler.GetArticlesByTagSlug)
		}

		// Homepage Sections công khai
		publicHomepageSections := public.Group("/homepage-sections")
		{
			publicHomepageSections.GET("", homepageSectionHandler.GetPublicSections)
			publicHomepageSections.GET("/:type_key", homepageSectionHandler.GetPublicSectionByTypeKey)
		}
	}
}
