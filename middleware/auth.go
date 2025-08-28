package middleware

import (
	"awesomeEval/internal/utils"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"net/http"
	"strings"
)

type AuthHandler struct {
	Redis *redis.Client
}

func NewAuthHandler(redisClient *redis.Client) *AuthHandler {
	return &AuthHandler{
		Redis: redisClient,
	}
}

func (h *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenValue := c.GetHeader("authorization")
		if tokenValue == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token required"})
			c.Abort()
			return
		}
		// remove bearer prefix
		tokenValue = strings.TrimPrefix(tokenValue, "Bearer ")
		tokenValue = strings.TrimSpace(tokenValue) // 避免多余空格
		// parse token
		claims := &jwt.MapClaims{}
		tkn, err := jwt.ParseWithClaims(tokenValue, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(utils.SecretKey), nil
		})
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		if tkn == nil || !tkn.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}
		// confirm token in redis
		val, err := h.Redis.Get(c, tokenValue).Result()
		if errors.Is(err, redis.Nil) || val == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
			c.Abort()
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		// set user_id to context
		c.Set("user_id", (*claims)["user_id"])
		c.Next()
	}
}
