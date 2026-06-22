package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const UserIDKey = "userID"
const UserEmailKey = "userEmail"

func JWTMiddleware(jwtSvc *JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Format: Bearer <token>",
			})
			return
		}
		claims, err := jwtSvc.ValidateToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			return
		}
		c.Set(UserIDKey, claims.UserID)
		c.Set(UserEmailKey, claims.Email)
		c.Next()
	}
}

func PrivateLockMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		pinToken := c.GetHeader("X-Pin-Token")
		if pinToken == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":  "PIN verification required",
				"action": "pin_required",
			})
			return
		}
		c.Set("pinToken", pinToken)
		c.Next()
	}
}