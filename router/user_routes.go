package router

import (
	"backend/internal/handle"
	"backend/utils"

	"github.com/gin-gonic/gin"
)

// SetupUserRoutes - Tất cả routes dành cho người dùng thông thường
func SetupUserRoutes(router *gin.Engine) {
	// Khởi tạo handlers
	categoryHandler := handle.NewCategoryHandler()
	productHandler := handle.NewProductHandler()
	orderHandler := handle.NewOrderHandler()
	newsHandler := handle.NewNewsHandler()
	cartHandler := handle.NewCartHandler()
	reviewHandler := handle.NewReviewHandler()
	discountHandler := handle.NewDiscountHandler()
	tagHandler := handle.NewTagHandler()
	newsCategoryHandler := handle.NewNewsCategoryHandler()
	addressHandler := handle.NewAddressHandler()
	s3Handler := handle.NewS3Handler()

	// Routes công khai - không cần xác thực
	public := router.Group("/api")
	{
		 // S3 Upload công khai
		public.POST("/upload/presigned", s3Handler.GetUploadUrl) // Presigned URL cho upload

		// Danh mục sản phẩm công khai
		publicCategories := public.Group("/categories")
		{
			publicCategories.GET("/", categoryHandler.GetCategories)
			publicCategories.GET("/:id", categoryHandler.GetCategoryByID)
			publicCategories.GET("/slug/:slug", categoryHandler.GetCategoryBySlug)
		}

		// Sản phẩm công khai
		publicProducts := public.Group("/products")
		{
			publicProducts.GET("/", productHandler.GetProducts)
			publicProducts.GET("/:id", productHandler.GetProductByID)
			publicProducts.GET("/sku/:sku", productHandler.GetProductBySKU)
		}

		// Đánh giá công khai
		publicReviews := public.Group("/reviews")
		{
			publicReviews.GET("/product/:product_id", reviewHandler.GetReviewsByProduct)
			publicReviews.GET("/product/:product_id/stats", reviewHandler.GetProductRatingStats)
			publicReviews.GET("/latest", reviewHandler.GetLatestReviews)
		}

		// Tin tức công khai
		publicNews := public.Group("/news")
		{
			publicNews.GET("/", newsHandler.GetNews)
			publicNews.GET("/:id", newsHandler.GetNewsByID)
			publicNews.GET("/slug/:slug", newsHandler.GetNewsBySlug)
		}

		// Danh mục tin tức công khai
		publicNewsCategories := public.Group("/news-categories")
		{
			publicNewsCategories.GET("/", newsCategoryHandler.GetNewsCategories)
			publicNewsCategories.GET("/tree", newsCategoryHandler.GetNewsCategoryTree)
			publicNewsCategories.GET("/:id", newsCategoryHandler.GetNewsCategoryByID)
			publicNewsCategories.GET("/slug/:slug", newsCategoryHandler.GetNewsCategoryBySlug)
		}

		// Tags công khai
		publicTags := public.Group("/tags")
		{
			publicTags.GET("/", tagHandler.GetTags)
			publicTags.GET("/popular", tagHandler.GetPopularTags)
			publicTags.GET("/search", tagHandler.SearchTags)
			publicTags.GET("/slug/:slug", tagHandler.GetTagBySlug)
			publicTags.GET("/:id", tagHandler.GetTagByID)
		}

		 // Mã giảm giá công khai
		publicDiscounts := public.Group("/discounts")
		{
			publicDiscounts.GET("/active", discountHandler.GetDiscounts) // Lấy mã giảm giá đang hoạt động
			publicDiscounts.POST("/validate", discountHandler.ValidateDiscount) // Kiểm tra mã giảm giá
		}

		// Địa chỉ công khai - chỉ có API lấy theo số điện thoại
		public.GET("/addresses", addressHandler.GetAddressesByPhone)

		// Đơn hàng công khai (cho khách vãng lai)
		publicOrders := public.Group("/public/orders")
		{
			publicOrders.POST("/", orderHandler.CreateOrder)
			publicOrders.GET("/track/:order_code", orderHandler.TrackOrderByNumber)
			publicOrders.POST("/lookup", orderHandler.LookupGuestOrders)
		}

		// Routes đơn hàng cơ bản (không cần auth)
		basicOrders := public.Group("/orders")
		{
			basicOrders.POST("/", orderHandler.CreateOrder)
			basicOrders.GET("/:id", orderHandler.GetOrderByID)
		}
	}

	// Routes cần xác thực - dành cho người dùng đã đăng nhập
	protected := router.Group("/api")
	protected.Use(utils.AuthMiddleware())
	{
		 // S3 Upload cho user đã xác thực
		protected.POST("/upload/presigned-auth", s3Handler.GetUploadUrl) // Presigned URL cho user đã xác thực
		protected.DELETE("/upload/delete", s3Handler.DeleteS3Object) // Xóa file

		 // Giỏ hàng
		cartRoutes := protected.Group("/cart")
		{
			cartRoutes.GET("/", cartHandler.GetCart)
			cartRoutes.POST("/add", cartHandler.AddToCart)
			cartRoutes.PUT("/items/:product_id", cartHandler.UpdateCartItem)
			cartRoutes.DELETE("/items/:product_id", cartHandler.RemoveFromCart)
			cartRoutes.DELETE("/clear", cartHandler.ClearCart)
		}

		// Đơn hàng của người dùng
		userOrders := protected.Group("/orders")
		{
			userOrders.GET("/my", orderHandler.GetMyOrders)
		}

		// Đánh giá của người dùng
		userReviews := protected.Group("/reviews")
		{
			userReviews.POST("/", reviewHandler.CreateReview)
			userReviews.GET("/my", reviewHandler.GetReviewsByUser)
			userReviews.PUT("/:id", reviewHandler.UpdateReview)
			userReviews.DELETE("/:id", reviewHandler.DeleteReview)
		}
	}
}
