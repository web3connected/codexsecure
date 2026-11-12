package z5

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Register wires system-execution zone routes.
func Register(group *gin.RouterGroup) {
	group.GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"zone":        "Z5",
			"zone_key":    "system",
			"description": "System zone route module is registered, but no concrete execution endpoints are exposed by codexsecure_api yet.",
			"status":      "module_registered",
			"endpoints":   []string{},
		})
	})
}
