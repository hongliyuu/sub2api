package admin

import (
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// LotteryHandler 管理端抽奖处理器
type LotteryHandler struct {
	lotteryService *service.LotteryService
	drawService    *service.LotteryDrawService
}

// NewLotteryHandler creates a new LotteryHandler
func NewLotteryHandler(
	lotteryService *service.LotteryService,
	drawService *service.LotteryDrawService,
) *LotteryHandler {
	return &LotteryHandler{
		lotteryService: lotteryService,
		drawService:    drawService,
	}
}

// Create 创建抽奖活动
// POST /api/v1/admin/lottery
func (h *LotteryHandler) Create(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req dto.CreateLotteryActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	input := service.CreateLotteryActivityInput{
		Title:                 req.Title,
		Description:           req.Description,
		ValidityDays:          req.ValidityDays,
		MinParticipants:       req.MinParticipants,
		BaseWinRate:           req.BaseWinRate,
		WinnerDiscountPercent: req.WinnerDiscountPercent,
		LoserCouponAmount:     req.LoserCouponAmount,
		AccountIDs:            req.AccountIDs,
		DailyLimitUSD:         req.DailyLimitUSD,
	}
	if req.DrawAt != nil {
		t := time.Unix(*req.DrawAt, 0)
		input.DrawAt = &t
	}

	activity, err := h.lotteryService.CreateActivity(c.Request.Context(), input, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.LotteryActivityFromService(activity))
}

// List 活动列表
// GET /api/v1/admin/lottery
func (h *LotteryHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	status := c.Query("status")

	activities, result, err := h.lotteryService.ListActivities(
		c.Request.Context(),
		pagination.PaginationParams{Page: page, PageSize: pageSize},
		status,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.LotteryActivity, 0, len(activities))
	for i := range activities {
		out = append(out, *dto.LotteryActivityFromService(&activities[i]))
	}
	response.Paginated(c, out, result.Total, page, pageSize)
}

// GetByID 活动详情
// GET /api/v1/admin/lottery/:id
func (h *LotteryHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid activity ID")
		return
	}

	activity, err := h.lotteryService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.LotteryActivityFromService(activity))
}

// Draw 手动开奖
// POST /api/v1/admin/lottery/:id/draw
func (h *LotteryHandler) Draw(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid activity ID")
		return
	}

	result, err := h.drawService.ExecuteDraw(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.LotteryDrawResultFromService(result))
}

// Cancel 取消活动
// DELETE /api/v1/admin/lottery/:id
func (h *LotteryHandler) Cancel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid activity ID")
		return
	}

	if err := h.lotteryService.CancelActivity(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "activity cancelled"})
}

// ListParticipants 参与者列表
// GET /api/v1/admin/lottery/:id/participants
func (h *LotteryHandler) ListParticipants(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid activity ID")
		return
	}

	page, pageSize := response.ParsePagination(c)
	participants, result, err := h.lotteryService.ListParticipants(
		c.Request.Context(),
		id,
		pagination.PaginationParams{Page: page, PageSize: pageSize},
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.LotteryParticipant, 0, len(participants))
	for i := range participants {
		out = append(out, *dto.LotteryParticipantFromService(&participants[i]))
	}
	response.Paginated(c, out, result.Total, page, pageSize)
}
