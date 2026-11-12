package z2

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Register wires identity-zone routes.
func Register(group *gin.RouterGroup) {
	group.GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"zone":        "Z2",
			"zone_key":    "identity",
			"description": "Identity zone route module is registered, but no concrete identity endpoints are exposed by codexsecure_api yet.",
			"status":      "module_registered",
			"endpoints":   []string{},
		})
	})
}
