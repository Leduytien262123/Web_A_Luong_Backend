package router

import (
	"backend/app"
	"backend/internal/handle"
	"backend/internal/repo"
	"backend/utils"

	"github.com/gin-gonic/gin"
)

// SetupAdminRoutes - Tất cả routes dành cho admin và owner
func SetupAdminRoutes(router *gin.Engine) {
	// Khởi tạo repositories và handlers
	dashboardHandler := handle.NewDashboardHandler()
	userRepo := repo.NewUserRepository(app.GetDB())
	adminHandler := handle.NewAdminHandler(userRepo)
	categoryHandler := handle.NewCategoryHandler()
	productHandler := handle.NewProductHandler()
	orderHandler := handle.NewOrderHandler()
	newsHandler := handle.NewNewsHandler()
	discountHandler := handle.NewDiscountHandler()
	reviewHandler := handle.NewReviewHandler()
	tagHandler := handle.NewTagHandler()
	newsCategoryHandler := handle.NewNewsCategoryHandler()
	s3Handler := handle.NewS3Handler()

	// Base admin group
	admin := router.Group("/api/admin")
	admin.Use(utils.AuthMiddleware())
	
	// Routes chỉ dành cho Owner
	ownerRoutes := admin.Group("/owner")
	ownerRoutes.Use(utils.OwnerMiddleware())
	{
		// Chỉ owner mới có thể xem thống kê hệ thống
		ownerRoutes.GET("/stats/system", adminHandler.GetUserStats)
	}

	// Routes dành cho Owner và Admin (cấp độ quản lý)
	managerRoutes := admin.Group("/manage")
	managerRoutes.Use(utils.AdminMiddleware())
	{
		// ===== DASHBOARD APIs - Tối ưu thành 3 API chính =====
		dashboardRoutes := managerRoutes.Group("/dashboard")
		{
			// API 1: Tổng quan - Gộp 6 API (overview, revenue-chart, order-status, top-products, top-categories, activities)
			dashboardRoutes.GET("/overview", dashboardHandler.GetFullOverview)
			
			// API 2: Phân tích chi tiết - Gộp 6 API (categories, products, payment-methods, order-types, customers, revenue-time)
			dashboardRoutes.GET("/analytics", dashboardHandler.GetAnalytics)
			
			// API 3: Cảnh báo - Gộp 3 API (low-stock, pending-orders, new-customers, activities, alert-counts)
			dashboardRoutes.GET("/alerts", dashboardHandler.GetAlerts)
		}

		// Quản lý người dùng
		managerRoutes.GET("/users", adminHandler.GetAllUsers)
		managerRoutes.POST("/user", adminHandler.CreateUser)
		managerRoutes.GET("/user/:id", adminHandler.GetUserByID)
		managerRoutes.PUT("/user/:id", adminHandler.UpdateUser)
		managerRoutes.GET("/user/role/:role", adminHandler.GetUsersByRole)
		managerRoutes.PUT("/user/:id/role", adminHandler.AssignUserRole)
		managerRoutes.PUT("/user/:id/status", adminHandler.ToggleUserStatus)
		managerRoutes.DELETE("/user/:id", adminHandler.DeleteUser)
		
		// Thống kê người dùng
		managerRoutes.GET("/stats/users", adminHandler.GetUserStats)

		// Quản lý danh mục sản phẩm
		managerRoutes.GET("/categories", categoryHandler.GetCategories) 
		managerRoutes.GET("/category/:id", categoryHandler.GetCategoryByID)
		managerRoutes.POST("/category", categoryHandler.CreateCategory)
		managerRoutes.PUT("/category/:id", categoryHandler.UpdateCategory)
		managerRoutes.DELETE("/category/:id", categoryHandler.DeleteCategory)

		// Quản lý sản phẩm
		managerRoutes.GET("/products", productHandler.GetProducts)
		managerRoutes.GET("/product/:id", productHandler.GetProductByID)
		managerRoutes.POST("/product", productHandler.CreateProduct)
		managerRoutes.PUT("/product/:id", productHandler.UpdateProduct)
		managerRoutes.DELETE("/product/:id", productHandler.DeleteProduct)
		managerRoutes.PATCH("/product/:id/stock", productHandler.UpdateProductStock)

		// Quản lý đơn hàng
		managerRoutes.GET("/orders", orderHandler.AdminGetOrders) // Thay đổi để sử dụng AdminGetOrders với bộ lọc
		managerRoutes.POST("/order", orderHandler.AdminCreateOrder) // Thay đổi để sử dụng AdminCreateOrder
		managerRoutes.GET("/order/stats", orderHandler.GetOrderStats)
		managerRoutes.GET("/order/guest-stats", orderHandler.GetGuestOrderStats)
		managerRoutes.GET("/order/:id", orderHandler.GetOrderByID)
		managerRoutes.PUT("/order/:id", orderHandler.AdminUpdateOrder) // Thêm route cập nhật đơn hàng
		managerRoutes.PUT("/order/:id/status", orderHandler.UpdateOrderStatus)
		managerRoutes.PUT("/order/:id/payment", orderHandler.UpdatePaymentStatus)
		managerRoutes.DELETE("/order/:id", orderHandler.AdminDeleteOrder) // Thêm route xóa đơn hàng

		// Quản lý mã giảm giá
		managerRoutes.GET("/discounts", discountHandler.GetDiscounts)
		managerRoutes.GET("/discount/:id", discountHandler.GetDiscountByID) // Chi tiết mã giảm giá theo ID
		managerRoutes.GET("/discount/code/:code", discountHandler.GetDiscountByCode) // Chi tiết mã giảm giá theo code
		managerRoutes.POST("/discount", discountHandler.CreateDiscount)
		managerRoutes.PUT("/discount/:id", discountHandler.UpdateDiscount)
		managerRoutes.DELETE("/discount/:id", discountHandler.DeleteDiscount)
		
		// API pause/resume mã giảm giá
		managerRoutes.PUT("/discount/:id/pause", discountHandler.PauseDiscount) // Tạm dừng mã giảm giá
		managerRoutes.PUT("/discount/:id/resume", discountHandler.ResumeDiscount) // Tiếp tục mã giảm giá
		
		// API mới cho hệ thống mã giảm giá nâng cao
		managerRoutes.POST("/discount/validate", discountHandler.ValidateDiscount) // Validate mã giảm giá
		managerRoutes.POST("/discount/apply", discountHandler.ApplyDiscountToOrder)
		managerRoutes.GET("/discount/product/:product_id", discountHandler.GetDiscountsByProduct)
		managerRoutes.GET("/discount/category/:category_id", discountHandler.GetDiscountsByCategory)

		// Quản lý tin tức - sửa route paths cho nhất quán
		managerRoutes.GET("/news", newsHandler.GetNews)
		managerRoutes.GET("/new/:id", newsHandler.GetNewsByID)  // Sửa từ "/new/:id"
		managerRoutes.GET("/new/slug/:slug", newsHandler.GetNewsBySlug)  // Thêm route get by slug
		managerRoutes.POST("/new", newsHandler.CreateNews)     // Sửa từ "/new"
		managerRoutes.PUT("/new/:id", newsHandler.UpdateNews)  // Sửa từ "/new/:id"
		managerRoutes.DELETE("/new/:id", newsHandler.DeleteNews) // Sửa từ "/new/:id"
		
		// Thêm các API đặc biệt cho tin tức
		managerRoutes.GET("/new/featured", newsHandler.GetFeaturedNews)
		managerRoutes.GET("/new/latest", newsHandler.GetLatestNews)
		managerRoutes.GET("/new/popular", newsHandler.GetPopularNews)
		managerRoutes.GET("/new/search", newsHandler.SearchNews)

		// Quản lý danh mục tin tức
		managerRoutes.GET("/news-categories", newsCategoryHandler.GetNewsCategories)
		managerRoutes.GET("/new-category/tree", newsCategoryHandler.GetNewsCategoryTree)
		managerRoutes.GET("/new-category/:id", newsCategoryHandler.GetNewsCategoryByID)
		managerRoutes.POST("/new-category", newsCategoryHandler.CreateNewsCategory)
		managerRoutes.PUT("/new-category/:id", newsCategoryHandler.UpdateNewsCategory)
		managerRoutes.DELETE("/new-category/:id", newsCategoryHandler.DeleteNewsCategory)

		// Quản lý tags
		managerRoutes.GET("/tags", tagHandler.GetTags)
		managerRoutes.GET("/tags/popular", tagHandler.GetPopularTags)
		managerRoutes.GET("/tag/search", tagHandler.SearchTags)
		managerRoutes.GET("/tag/slug/:slug", tagHandler.GetTagBySlug)
		managerRoutes.GET("/tag/:id", tagHandler.GetTagByID)
		managerRoutes.POST("/tag", tagHandler.CreateTag)
		managerRoutes.PUT("/tag/:id", tagHandler.UpdateTag)
		managerRoutes.DELETE("/tag/:id", tagHandler.DeleteTag)

		// Quản lý đánh giá
		managerRoutes.GET("/reviews/latest", reviewHandler.GetLatestReviews)
		managerRoutes.GET("/review/product/:product_id", reviewHandler.GetReviewsByProduct)
		managerRoutes.PUT("/review/:id/toggle", reviewHandler.AdminToggleReviewStatus)

		 // S3 Upload và Storage Management cho admin
		managerRoutes.POST("/upload/s3", s3Handler.GetUploadUrl) // Tạo presigned URL để upload
		managerRoutes.DELETE("/upload/delete", s3Handler.DeleteS3Object) // Xóa file từ S3
		managerRoutes.GET("/storage/usage", s3Handler.GetS3BucketMemoryUsage) // Lấy thông tin dung lượng
	}
}
