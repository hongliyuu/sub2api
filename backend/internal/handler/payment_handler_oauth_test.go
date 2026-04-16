package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	paymentcore "github.com/Wei-Shaw/sub2api/internal/payment"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPaymentHandler_CreateOrderReturnsOAuthRequiredForWeChatInApp(t *testing.T) {
	gin.SetMode(gin.TestMode)

	settingSvc := service.NewSettingService(&settingHandlerRepoStub{values: map[string]string{}}, &config.Config{
		WeChat: config.WeChatConnectConfig{
			Enabled:             true,
			AppID:               "wx123456",
			AppSecret:           "wechat-secret",
			Scopes:              "snsapi_login",
			RedirectURL:         "https://example.com/api/v1/auth/oauth/wechat/payment/callback",
			FrontendRedirectURL: "/auth/wechat/callback",
		},
	})
	paymentSvc := service.NewPaymentService(nil, nil, nil, nil, nil, nil, settingSvc, nil, nil)
	paymentHandler := NewPaymentHandler(paymentSvc, nil, nil, settingSvc)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := strings.NewReader(`{"amount":19.9,"payment_type":"wxpay","order_type":"balance"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment/orders", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 MicroMessenger/8.0.49")
	req.Header.Set("Referer", "https://app.example.com/purchase?plan=starter")
	c.Request = req
	c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 7, Concurrency: 1})

	paymentHandler.CreateOrder(c)
	require.Equal(t, http.StatusOK, rec.Code)

	var bodyResp struct {
		Code int `json:"code"`
		Data struct {
			ResultType  string  `json:"result_type"`
			PaymentType string  `json:"payment_type"`
			Amount      float64 `json:"amount"`
			OAuth       struct {
				AuthorizeURL string `json:"authorize_url"`
				AppID        string `json:"appid"`
				Scope        string `json:"scope"`
				RedirectURL  string `json:"redirect_url"`
			} `json:"oauth"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &bodyResp))
	require.Equal(t, 0, bodyResp.Code)
	require.Equal(t, string(paymentcore.CreatePaymentResultOAuthRequired), bodyResp.Data.ResultType)
	require.Equal(t, paymentcore.TypeWxpay, bodyResp.Data.PaymentType)
	require.Equal(t, 19.9, bodyResp.Data.Amount)
	require.Equal(t, "wx123456", bodyResp.Data.OAuth.AppID)
	require.Equal(t, "snsapi_base", bodyResp.Data.OAuth.Scope)
	require.Equal(t, "/auth/wechat/callback", bodyResp.Data.OAuth.RedirectURL)

	startURL, err := url.Parse(bodyResp.Data.OAuth.AuthorizeURL)
	require.NoError(t, err)
	require.Empty(t, startURL.Scheme)
	require.Empty(t, startURL.Host)
	require.Equal(t, "/api/v1/auth/oauth/wechat/payment/start", startURL.Path)
	require.Equal(t, "wxpay", startURL.Query().Get("payment_type"))
	require.Equal(t, "19.9", startURL.Query().Get("amount"))
	require.Equal(t, "balance", startURL.Query().Get("order_type"))
	require.Equal(t, "/purchase?plan=starter", startURL.Query().Get("redirect"))
}
