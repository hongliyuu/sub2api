package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// LotteryHandler 用户端抽奖处理器
type LotteryHandler struct {
	lotteryService *service.LotteryService
	drawService    *service.LotteryDrawService
}

// NewLotteryHandler creates a new user-facing LotteryHandler
func NewLotteryHandler(
	lotteryService *service.LotteryService,
	drawService *service.LotteryDrawService,
) *LotteryHandler {
	return &LotteryHandler{
		lotteryService: lotteryService,
		drawService:    drawService,
	}
}

// GetActive 获取进行中的活动（返回数组）
// GET /api/v1/lottery/active
func (h *LotteryHandler) GetActive(c *gin.Context) {
	activity, err := h.lotteryService.GetActiveActivity(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// 返回数组以匹配前端契约
	out := make([]dto.LotteryActivity, 0, 1)
	if activity != nil {
		out = append(out, *dto.LotteryActivityFromService(activity))
	}
	response.Success(c, out)
}

// GetMyParticipations 我参与的活动
// GET /api/v1/lottery/my
func (h *LotteryHandler) GetMyParticipations(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	page, pageSize := response.ParsePagination(c)
	participants, result, err := h.lotteryService.ListUserParticipations(
		c.Request.Context(),
		subject.UserID,
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

// GetMyCoupons 我的优惠券
// GET /api/v1/lottery/coupons
func (h *LotteryHandler) GetMyCoupons(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	page, pageSize := response.ParsePagination(c)
	status := c.Query("status")

	coupons, result, err := h.lotteryService.ListUserCoupons(
		c.Request.Context(),
		subject.UserID,
		status,
		pagination.PaginationParams{Page: page, PageSize: pageSize},
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.LotteryCoupon, 0, len(coupons))
	for i := range coupons {
		out = append(out, *dto.LotteryCouponFromService(&coupons[i]))
	}
	response.Paginated(c, out, result.Total, page, pageSize)
}

// GetByID 活动详情
// GET /api/v1/lottery/:id
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

// Participate 参与抽奖
// POST /api/v1/lottery/:id/participate
func (h *LotteryHandler) Participate(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid activity ID")
		return
	}

	participant, err := h.drawService.Participate(c.Request.Context(), id, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.LotteryParticipantFromService(participant))
}

// GetByShareCode 公开分享页数据
// GET /api/v1/lottery/share/:code
func (h *LotteryHandler) GetByShareCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		response.BadRequest(c, "Share code is required")
		return
	}

	activity, err := h.lotteryService.GetByShareCode(c.Request.Context(), code)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.LotteryActivityFromService(activity))
}
