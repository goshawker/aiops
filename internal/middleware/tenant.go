package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// TenantMiddleware extracts tenant context from request headers.
func TenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract tenant ID from header (set by auth middleware or gateway)
		tenantIDStr := c.GetHeader("X-Tenant-ID")
		if tenantIDStr == "" {
			tenantIDStr = "1" // Default tenant
		}

		tenantID, err := strconv.ParseInt(tenantIDStr, 10, 64)
		if err != nil || tenantID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效租户"})
			c.Abort()
			return
		}

		c.Set("tenant_id", tenantID)
		c.Next()
	}
}

// GetTenantID extracts tenant ID from gin context.
func GetTenantID(c *gin.Context) int64 {
	id, exists := c.Get("tenant_id")
	if !exists {
		return 1
	}
	return id.(int64)
}
