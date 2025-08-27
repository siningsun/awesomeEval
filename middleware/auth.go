package middleware

import (
	"awesomeEval/internal/utils"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"net/http"
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
		}
		claims := &jwt.MapClaims{}
		tkn, err := jwt.ParseWithClaims(tokenValue, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(utils.SecretKey), nil
		})
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		}
		if tkn == nil || !tkn.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		}
		// confirm token in redis
		val, err := h.Redis.Get(c, tokenValue).Result()
		if errors.Is(err, redis.Nil) || val == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		// set user_id to context
		c.Set("user_id", (*claims)["user_id"])
		c.Next()
	}
}
