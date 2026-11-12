package routes

import "github.com/gin-gonic/gin"

// LoadCodexZones is retained as a compatibility alias.
// Canonical API route definitions now live in api.go.
func LoadCodexZones(v1 *gin.RouterGroup, deps Dependencies) {
	LoadAPIRoutes(v1, deps)
}
