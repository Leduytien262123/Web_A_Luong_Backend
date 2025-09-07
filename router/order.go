package router

import (
	"backend/internal/handle"
	"backend/utils"

	"github.com/gin-gonic/gin"
)

func SetupOrderRoutes(r *gin.Engine) {
	orderHandler := handle.NewOrderHandler()

	publicRoutes := r.Group("/api/public/orders")
	{
		publicRoutes.POST("/", orderHandler.CreateOrder)
		publicRoutes.GET("/track/:order_number", orderHandler.TrackOrderByNumber)
		publicRoutes.POST("/lookup", orderHandler.LookupGuestOrders)
	}

	orderRoutes := r.Group("/api/orders")
	{
		orderRoutes.POST("/", orderHandler.CreateOrder)
		orderRoutes.GET("/:id", orderHandler.GetOrderByID)
	}

	userRoutes := r.Group("/api/orders")
	userRoutes.Use(utils.AuthMiddleware())
	{
		userRoutes.GET("/my", orderHandler.GetMyOrders)
	}

	adminRoutes := r.Group("/api/admin/manage")
	adminRoutes.Use(utils.AuthMiddleware())
	adminRoutes.Use(utils.AdminMiddleware())
	{
		adminRoutes.GET("/orders", orderHandler.GetOrders)
		adminRoutes.GET("/orders/stats", orderHandler.GetOrderStats)
		adminRoutes.GET("/orders/guest-stats", orderHandler.GetGuestOrderStats)
		adminRoutes.GET("/orders/:id", orderHandler.GetOrderByID)
		adminRoutes.PUT("/orders/:id/status", orderHandler.UpdateOrderStatus)
		adminRoutes.PUT("/orders/:id/payment", orderHandler.UpdatePaymentStatus)
	}
}