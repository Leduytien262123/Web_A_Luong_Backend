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
		// Quản lý người dùng
		managerRoutes.POST("/users", adminHandler.CreateUser)
		managerRoutes.GET("/users", adminHandler.GetAllUsers)
		managerRoutes.GET("/users/:id", adminHandler.GetUserByID)
		managerRoutes.PUT("/users/:id", adminHandler.UpdateUser)
		managerRoutes.GET("/users/role/:role", adminHandler.GetUsersByRole)
		managerRoutes.PUT("/users/:id/role", adminHandler.AssignUserRole)
		managerRoutes.PUT("/users/:id/status", adminHandler.ToggleUserStatus)
		managerRoutes.DELETE("/users/:id", adminHandler.DeleteUser)
		
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
		managerRoutes.GET("/discount/:code", discountHandler.GetDiscountByCode)
		managerRoutes.POST("/discount", discountHandler.CreateDiscount)
		managerRoutes.PUT("/discount/:id", discountHandler.UpdateDiscount)
		managerRoutes.DELETE("/discount/:id", discountHandler.DeleteDiscount)

		// Quản lý tin tức
		managerRoutes.GET("/news", newsHandler.GetNews)
		managerRoutes.GET("/new/:id", newsHandler.GetNewsByID)
		managerRoutes.POST("/new", newsHandler.CreateNews)
		managerRoutes.PUT("/new/:id", newsHandler.UpdateNews)
		managerRoutes.DELETE("/new/:id", newsHandler.DeleteNews)

		// Quản lý danh mục tin tức
		managerRoutes.GET("/news-categories", newsCategoryHandler.GetNewsCategories)
		managerRoutes.GET("/new-category/tree", newsCategoryHandler.GetNewsCategoryTree)
		managerRoutes.GET("/new-category/:id", newsCategoryHandler.GetNewsCategoryByID)
		managerRoutes.POST("/new-category", newsCategoryHandler.CreateNewsCategory)
		managerRoutes.PUT("/new-category/:id", newsCategoryHandler.UpdateNewsCategory)
		managerRoutes.DELETE("/new-category/:id", newsCategoryHandler.DeleteNewsCategory)

		// Quản lý tags
		managerRoutes.GET("/tags", tagHandler.GetTags)
		managerRoutes.GET("/tag/search", tagHandler.SearchTags)
		managerRoutes.GET("/tag/:id", tagHandler.GetTagByID)
		managerRoutes.POST("/tag", tagHandler.CreateTag)
		managerRoutes.PUT("/tag/:id", tagHandler.UpdateTag)
		managerRoutes.DELETE("/tag/:id", tagHandler.DeleteTag)

		// Quản lý đánh giá
		managerRoutes.GET("/reviews/latest", reviewHandler.GetLatestReviews)
		managerRoutes.GET("/review/product/:product_id", reviewHandler.GetReviewsByProduct)
		managerRoutes.PUT("/review/:id/toggle", reviewHandler.AdminToggleReviewStatus)
	}
}
