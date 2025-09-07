package router

import (
	"backend/app"
	"backend/internal/handle"
	"backend/internal/repo"
	"backend/utils"

	"github.com/gin-gonic/gin"
)

func SetupAdminRoutes(router *gin.Engine) {
	userRepo := repo.NewUserRepository(app.GetDB())
	adminHandler := handle.NewAdminHandler(userRepo)

	// Các routes admin với các cấp độ quyền khác nhau
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
	managerRoutes.Use(utils.OwnerOrAdminMiddleware())
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
		
		 // Thống kê
		managerRoutes.GET("/stats/users", adminHandler.GetUserStats)
	}
}