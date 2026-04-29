package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// add unique request id in header
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid := uuid.NewString()

		c.Request.Header.Set("X-Request-ID", uuid)
		c.Next()
	}
}
