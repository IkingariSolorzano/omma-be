package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/IkingariSolorzano/omma-be/models"
)

type Claims struct {
	UserID uint             `json:"user_id"`
	Email  string           `json:"email"`
	Role   models.UserRole  `json:"role"`
	jwt.RegisteredClaims
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Debug logging
		fmt.Printf("[AUTH] Request: %s %s\n", c.Request.Method, c.Request.URL.Path)
		
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			fmt.Printf("[AUTH] Missing Authorization header\n")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token de autorización requerido"})
			c.Abort()
			return
		}

		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
		
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			fmt.Printf("[AUTH] Token validation failed: %v\n", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(*Claims); ok {
			fmt.Printf("[AUTH] Token valid for user ID: %d\n", claims.UserID)
			c.Set("user_id", claims.UserID)
			c.Set("user_email", claims.Email)
			c.Set("user_role", claims.Role)
		}

		fmt.Printf("[AUTH] Authentication successful, proceeding to handler\n")
		c.Next()
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists || role != models.RoleAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Acceso denegado"})
			c.Abort()
			return
		}
		c.Next()
	}
}
