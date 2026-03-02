package routes

import (
	"github.com/Wei-Shaw/nbapi/internal/handler"
	"github.com/Wei-Shaw/nbapi/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterProvisionRoutes registers the /api/v1/provision endpoint
func RegisterProvisionRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	serviceTokenAuth middleware.ServiceTokenAuthMiddleware,
) {
	provision := v1.Group("")
	provision.Use(gin.HandlerFunc(serviceTokenAuth))
	{
		provision.POST("/provision", h.Provision.Provision)
	}
}
