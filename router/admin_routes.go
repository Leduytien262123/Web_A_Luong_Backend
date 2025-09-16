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

		// Quản lý danh mục
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
		managerRoutes.GET("/orders", orderHandler.GetOrders)
		managerRoutes.POST("/orders", orderHandler.CreateOrder)
		managerRoutes.GET("/orders/stats", orderHandler.GetOrderStats)
		managerRoutes.GET("/orders/guest-stats", orderHandler.GetGuestOrderStats)
		managerRoutes.GET("/orders/:id", orderHandler.GetOrderByID)
		managerRoutes.PUT("/orders/:id/status", orderHandler.UpdateOrderStatus)
		managerRoutes.PUT("/orders/:id/payment", orderHandler.UpdatePaymentStatus)

		// Quản lý thẻ giảm giá
		managerRoutes.GET("/discounts", discountHandler.GetDiscounts)
		managerRoutes.GET("/discounts/:code", discountHandler.GetDiscountByCode)
		managerRoutes.POST("/discounts", discountHandler.CreateDiscount)
		managerRoutes.PUT("/discounts/:id", discountHandler.UpdateDiscount)
		managerRoutes.DELETE("/discounts/:id", discountHandler.DeleteDiscount)

		// Quản lý tin tức
		managerRoutes.GET("/news", newsHandler.GetNews)
		managerRoutes.GET("/new/:id", newsHandler.GetNewsByID)
		managerRoutes.POST("/new", newsHandler.CreateNews)
		managerRoutes.PUT("/new/:id", newsHandler.UpdateNews)
		managerRoutes.DELETE("/new/:id", newsHandler.DeleteNews)

		// Quản lý đánh giá
		managerRoutes.PUT("/reviews/:id/toggle", reviewHandler.AdminToggleReviewStatus)
	}
}
