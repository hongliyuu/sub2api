package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type redeemStatsRepoStub struct {
	service.RedeemCodeRepository
	stats *service.RedeemCodeStats
	err   error
}

func (s *redeemStatsRepoStub) GetStats(ctx context.Context) (*service.RedeemCodeStats, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.stats, nil
}

func TestRedeemHandlerGetStats_ReturnsRealStats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	redeemSvc := service.NewRedeemService(&redeemStatsRepoStub{
		stats: &service.RedeemCodeStats{
			TotalCodes:            12,
			UnusedCodes:           7,
			UsedCodes:             3,
			ExpiredCodes:          2,
			TotalValueDistributed: 88.5,
			TypeCounts: map[string]int64{
				service.RedeemTypeBalance:      5,
				service.RedeemTypeConcurrency:  3,
				service.RedeemTypeSubscription: 2,
				service.RedeemTypeInvitation:   2,
			},
		},
	}, nil, nil, nil, nil, nil, nil)

	handler := NewRedeemHandler(newStubAdminService(), redeemSvc)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/redeem-codes/stats", nil)

	handler.GetStats(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			TotalCodes            int64            `json:"total_codes"`
			ActiveCodes           int64            `json:"active_codes"`
			UsedCodes             int64            `json:"used_codes"`
			ExpiredCodes          int64            `json:"expired_codes"`
			TotalValueDistributed float64          `json:"total_value_distributed"`
			ByType                map[string]int64 `json:"by_type"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, int64(12), resp.Data.TotalCodes)
	require.Equal(t, int64(7), resp.Data.ActiveCodes)
	require.Equal(t, int64(3), resp.Data.UsedCodes)
	require.Equal(t, int64(2), resp.Data.ExpiredCodes)
	require.Equal(t, 88.5, resp.Data.TotalValueDistributed)
	require.Equal(t, int64(5), resp.Data.ByType[service.RedeemTypeBalance])
	require.Equal(t, int64(3), resp.Data.ByType[service.RedeemTypeConcurrency])
	require.Equal(t, int64(2), resp.Data.ByType[service.RedeemTypeSubscription])
	require.Equal(t, int64(2), resp.Data.ByType[service.RedeemTypeInvitation])
}

func TestRedeemHandlerGetStats_WithoutRedeemServiceReturns500(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewRedeemHandler(newStubAdminService(), nil)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/redeem-codes/stats", nil)

	handler.GetStats(c)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}
