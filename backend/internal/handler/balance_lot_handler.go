package handler

import (
	"strconv"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// BalanceLotHandler 余额批次处理器
type BalanceLotHandler struct {
	balanceLotService *service.BalanceLotService
}

// NewBalanceLotHandler 创建余额批次处理器
func NewBalanceLotHandler(balanceLotService *service.BalanceLotService) *BalanceLotHandler {
	return &BalanceLotHandler{balanceLotService: balanceLotService}
}

// List 用户查看自己的余额批次列表
// GET /api/v1/user/balance-lots
func (h *BalanceLotHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	page, pageSize := response.ParsePagination(c)
	status := c.Query("status")

	lots, total, err := h.balanceLotService.GetUserLots(c.Request.Context(), subject.UserID, status, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Paginated(c, lots, total, page, pageSize)
}

// Summary 用户余额概览
// GET /api/v1/user/balance-lots/summary
func (h *BalanceLotHandler) Summary(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	summary, err := h.balanceLotService.GetUserSummary(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, summary)
}

// AdminListUserLots 管理员查看用户余额批次
// GET /api/v1/admin/users/:id/balance-lots
func (h *BalanceLotHandler) AdminListUserLots(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	page, pageSize := response.ParsePagination(c)
	status := c.Query("status")

	lots, total, err := h.balanceLotService.GetUserLots(c.Request.Context(), userID, status, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Paginated(c, lots, total, page, pageSize)
}
