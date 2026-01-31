package webhook

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// mockPaymentCallbackRepository 模拟支付回调仓库
type mockPaymentCallbackRepository struct {
	createCalled bool
	updateCalled bool
	lastRecord   *repository.PaymentCallbackRecord
	createdID    int64
}

func (m *mockPaymentCallbackRepository) Create(_ any, record *repository.PaymentCallbackRecord) (*ent.PaymentCallback, error) {
	m.createCalled = true
	m.lastRecord = record
	return &ent.PaymentCallback{ID: m.createdID}, nil
}

func (m *mockPaymentCallbackRepository) Update(_ any, _ int64, record *repository.PaymentCallbackRecord) (*ent.PaymentCallback, error) {
	m.updateCalled = true
	m.lastRecord = record
	return &ent.PaymentCallback{ID: m.createdID}, nil
}

func TestHandlePaymentNotify_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Response format should match WeChat Pay spec", func(t *testing.T) {
		// 检查成功响应格式
		response := WeChatPayResponse{
			Code:    "SUCCESS",
			Message: "",
		}
		data, _ := json.Marshal(response)
		assert.Equal(t, `{"code":"SUCCESS","message":""}`, string(data))

		// 检查失败响应格式
		failResponse := WeChatPayResponse{
			Code:    "FAIL",
			Message: "签名验证失败",
		}
		failData, _ := json.Marshal(failResponse)
		assert.Equal(t, `{"code":"FAIL","message":"签名验证失败"}`, string(failData))
	})
}

func TestWeChatPayWebhookHandler_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("respondSuccess returns correct format", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		h := &WeChatPayWebhookHandler{}
		h.respondSuccess(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response WeChatPayResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "SUCCESS", response.Code)
		assert.Equal(t, "", response.Message)
	})

	t.Run("respondFail returns correct format", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		h := &WeChatPayWebhookHandler{}
		h.respondFail(c, "测试错误")

		assert.Equal(t, http.StatusOK, w.Code)

		var response WeChatPayResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "FAIL", response.Code)
		assert.Equal(t, "测试错误", response.Message)
	})
}

func TestWeChatPayWebhookHandler_RequestBodyParsing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("can read request body", func(t *testing.T) {
		body := `{"resource":{"ciphertext":"encrypted_data"}}`
		req := httptest.NewRequest(http.MethodPost, "/webhook/wechat/payment", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Wechatpay-Timestamp", "1234567890")
		req.Header.Set("Wechatpay-Nonce", "nonce123")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		// 读取请求体
		bodyBytes := make([]byte, len(body))
		n, err := c.Request.Body.Read(bodyBytes)
		if err != nil && err.Error() != "EOF" {
			t.Fatalf("failed to read body: %v", err)
		}
		assert.Equal(t, len(body), n)
		assert.Equal(t, body, string(bodyBytes))
	})
}

func TestExtractWeChatPayHeaders(t *testing.T) {
	t.Run("extracts WeChat Pay headers correctly", func(t *testing.T) {
		header := make(http.Header)
		header.Set("Wechatpay-Timestamp", "1234567890")
		header.Set("Wechatpay-Nonce", "nonce123")
		header.Set("Wechatpay-Signature", "signature456")
		header.Set("Wechatpay-Serial", "serial789")
		header.Set("Content-Type", "application/json")
		header.Set("X-Custom-Header", "should-be-ignored")

		headers := extractWeChatPayHeaders(header)

		assert.Equal(t, "1234567890", headers["Wechatpay-Timestamp"])
		assert.Equal(t, "nonce123", headers["Wechatpay-Nonce"])
		assert.Equal(t, "signature456", headers["Wechatpay-Signature"])
		assert.Equal(t, "serial789", headers["Wechatpay-Serial"])
		assert.Equal(t, "application/json", headers["Content-Type"])
		assert.Empty(t, headers["X-Custom-Header"])
	})
}

func TestSafeStringPtr(t *testing.T) {
	t.Run("returns empty for nil", func(t *testing.T) {
		assert.Equal(t, "", safeStringPtr(nil))
	})

	t.Run("returns value for non-nil", func(t *testing.T) {
		s := "test"
		assert.Equal(t, "test", safeStringPtr(&s))
	})
}

func TestGetCallbackID(t *testing.T) {
	t.Run("returns 0 for nil", func(t *testing.T) {
		assert.Equal(t, int64(0), getCallbackID(nil))
	})

	t.Run("returns ID for non-nil", func(t *testing.T) {
		callback := &ent.PaymentCallback{ID: 123}
		assert.Equal(t, int64(123), getCallbackID(callback))
	})
}

func TestValidateTimestamp(t *testing.T) {
	t.Run("returns error for empty timestamp", func(t *testing.T) {
		err := service.ValidateTimestamp("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("returns error for invalid format", func(t *testing.T) {
		err := service.ValidateTimestamp("not-a-number")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid timestamp format")
	})

	t.Run("returns error for expired timestamp", func(t *testing.T) {
		// 很久以前的时间戳
		expiredTimestamp := "1000000000"
		err := service.ValidateTimestamp(expiredTimestamp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("accepts valid timestamp", func(t *testing.T) {
		// 当前时间戳
		currentTimestamp := strconv.FormatInt(time.Now().Unix(), 10)
		err := service.ValidateTimestamp(currentTimestamp)
		assert.NoError(t, err)
	})

	t.Run("accepts timestamp within 5 minutes", func(t *testing.T) {
		// 4分钟前的时间戳（应该通过）
		pastTimestamp := strconv.FormatInt(time.Now().Add(-4*time.Minute).Unix(), 10)
		err := service.ValidateTimestamp(pastTimestamp)
		assert.NoError(t, err)
	})

	t.Run("rejects timestamp older than 5 minutes", func(t *testing.T) {
		// 6分钟前的时间戳（应该失败）
		oldTimestamp := strconv.FormatInt(time.Now().Add(-6*time.Minute).Unix(), 10)
		err := service.ValidateTimestamp(oldTimestamp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})
}

func TestPaymentNotifyResult(t *testing.T) {
	t.Run("IsValid returns false when SignatureErr is set", func(t *testing.T) {
		result := &service.PaymentNotifyResult{
			SignatureErr: assert.AnError,
		}
		assert.False(t, result.IsValid())
	})

	t.Run("IsValid returns false when TimestampErr is set", func(t *testing.T) {
		result := &service.PaymentNotifyResult{
			TimestampErr: assert.AnError,
		}
		assert.False(t, result.IsValid())
	})

	t.Run("IsValid returns false when DecryptionErr is set", func(t *testing.T) {
		result := &service.PaymentNotifyResult{
			DecryptionErr: assert.AnError,
		}
		assert.False(t, result.IsValid())
	})

	t.Run("IsValid returns false when Transaction is nil", func(t *testing.T) {
		result := &service.PaymentNotifyResult{}
		assert.False(t, result.IsValid())
	})

	t.Run("Error returns first error", func(t *testing.T) {
		result := &service.PaymentNotifyResult{
			SignatureErr: assert.AnError,
		}
		assert.Equal(t, assert.AnError, result.Error())
	})
}
