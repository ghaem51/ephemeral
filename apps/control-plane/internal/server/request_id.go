package server

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const requestIDKey = "requestID"

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			bytes := make([]byte, 12)
			if _, err := rand.Read(bytes); err == nil {
				id = hex.EncodeToString(bytes)
			} else {
				id = "unavailable"
			}
		}
		c.Set(requestIDKey, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func requestID(c *gin.Context) string {
	id, _ := c.Get(requestIDKey)
	value, _ := id.(string)
	return value
}
