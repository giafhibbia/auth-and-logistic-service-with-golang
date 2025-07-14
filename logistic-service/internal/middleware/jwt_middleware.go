package middleware

import (
    "github.com/gin-gonic/gin"
    "logistic-service/internal/service"
    "net/http"
    "strings"
)

// JWTAuthMiddleware validates the JWT token from the Authorization header.
// It expects the header to be in the format: "Bearer <token>".
// If valid, it stores the claims in the context for downstream handlers.
func JWTAuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")

        if !strings.HasPrefix(authHeader, "Bearer ") {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid Authorization header"})
            c.Abort()
            return
        }

        tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
        claims, err := service.ParseJWT(tokenStr)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired JWT"})
            c.Abort()
            return
        }

        // Save claims into context for use in handlers
        c.Set("claims", claims)
        c.Next()
    }
}
