package utils

import (
	"backend/internal/consts"
	"backend/internal/helpers"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Auth middleware để xác thực JWT token
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			helpers.UnauthorizedResponse(c, consts.MSG_UNAUTHORIZED)
			c.Abort()
			return
		}

		// Kiểm tra định dạng Bearer token
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			helpers.UnauthorizedResponse(c, consts.MSG_UNAUTHORIZED)
			c.Abort()
			return
		}

		token, err := helpers.ValidateJWT(tokenParts[1])
		if err != nil || !token.Valid {
			helpers.UnauthorizedResponse(c, consts.MSG_UNAUTHORIZED)
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			helpers.UnauthorizedResponse(c, consts.MSG_UNAUTHORIZED)
			c.Abort()
			return
		}

		// Parse và set user info
		userIDStr, ok := claims["user_id"].(string)
		if !ok {
			helpers.UnauthorizedResponse(c, consts.MSG_UNAUTHORIZED)
			c.Abort()
			return
		}
		
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			helpers.UnauthorizedResponse(c, consts.MSG_UNAUTHORIZED)
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("username", claims["username"].(string))
		c.Set("user_role", claims["role"].(string))
		c.Next()
	}
}

// CORS middleware để xử lý cross-origin requests
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Set CORS headers
		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight OPTIONS request
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}

// Generic role middleware
func roleMiddleware(allowedRoles []string, message string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists {
			helpers.ForbiddenResponse(c, consts.MSG_FORBIDDEN)
			c.Abort()
			return
		}

		roleStr := role.(string)
		for _, allowedRole := range allowedRoles {
			if roleStr == allowedRole {
				c.Next()
				return
			}
		}

		helpers.ForbiddenResponse(c, message)
		c.Abort()
	}
}

// Admin middleware (admin hoặc owner)
func AdminMiddleware() gin.HandlerFunc {
	return roleMiddleware([]string{consts.RoleAdmin, consts.RoleOwner}, consts.MSG_FORBIDDEN)
}

// Owner middleware (chỉ owner)
func OwnerMiddleware() gin.HandlerFunc {
	return roleMiddleware([]string{consts.RoleOwner}, "Yêu cầu quyền truy cập Owner")
}

// Member middleware (member, admin hoặc owner)
func MemberOrAboveMiddleware() gin.HandlerFunc {
	return roleMiddleware([]string{"member", consts.RoleAdmin, consts.RoleOwner}, "Yêu cầu quyền truy cập Member trở lên")
}
