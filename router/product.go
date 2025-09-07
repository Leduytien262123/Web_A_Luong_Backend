package router

import (
	"backend/internal/handle"
	"backend/utils"

	"github.com/gin-gonic/gin"
)

func SetupProductRoutes(r *gin.Engine) {
	productHandler := handle.NewProductHandler()

	publicRoutes := r.Group("/api/products")
	{
		publicRoutes.GET("/", productHandler.GetProducts)
		publicRoutes.GET("/:id", productHandler.GetProductByID)
		publicRoutes.GET("/sku/:sku", productHandler.GetProductBySKU)
	}

	adminRoutes := r.Group("/api/admin/manage")
	adminRoutes.Use(utils.AuthMiddleware())
	adminRoutes.Use(utils.AdminMiddleware())
	{
		adminRoutes.GET("/products", productHandler.GetProducts)
		adminRoutes.GET("/product/:id", productHandler.GetProductByID)
		adminRoutes.POST("/product", productHandler.CreateProduct)
		adminRoutes.PUT("/product/:id", productHandler.UpdateProduct)
		adminRoutes.DELETE("/product/:id", productHandler.DeleteProduct)
		adminRoutes.PATCH("/product/:id/stock", productHandler.UpdateProductStock)
	}
}
