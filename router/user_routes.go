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

	// Routes công khai - không cần xác thực
	public := router.Group("/api")
	{
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

		// Tin tức công khai
		publicNews := public.Group("/news")
		{
			publicNews.GET("/", newsHandler.GetNews)
			publicNews.GET("/:id", newsHandler.GetNewsByID)
			publicNews.GET("/slug/:slug", newsHandler.GetNewsBySlug)
		}

		// Đơn hàng công khai (cho khách vãng lai)
		publicOrders := public.Group("/public/orders")
		{
			publicOrders.POST("/", orderHandler.CreateOrder)
			publicOrders.GET("/track/:order_number", orderHandler.TrackOrderByNumber)
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
	}
}
