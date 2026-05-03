package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"anthill/admin/internal/database"
)

func AuditLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		bodyBytes, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		c.Next()

		userIDVal, _ := c.Get("user_id")
		usernameVal, _ := c.Get("username")

		userID := int64(0)
		if uid, ok := userIDVal.(int); ok {
			userID = int64(uid)
		}

		username := ""
		if u, ok := usernameVal.(string); ok {
			username = u
		}

		action := c.Request.Method + " " + c.FullPath()
		if action == "" {
			action = c.Request.Method + " " + c.Request.URL.Path
		}

		status := c.Writer.Status()

		details := ""
		if len(bodyBytes) > 0 && len(bodyBytes) < 500 {
			details = string(bodyBytes)
		}

		database.LogAudit(
			userID,
			username,
			action,
			c.Request.URL.Path,
			c.Request.Method,
			c.ClientIP(),
			status,
			details,
		)

		_ = start
	}
}