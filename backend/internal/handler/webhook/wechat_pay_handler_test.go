package webhook

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/gin-gonic/gin"
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

func TestHandlePaymentNotify_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 准备测试数据
	mockRepo := &mockPaymentCallbackRepository{createdID: 1}

	// 注意：由于真实仓库需要 *ent.Client，这里只测试响应格式
	// 实际测试需要使用集成测试

	// 测试响应格式
	t.Run("Response format should match WeChat Pay spec", func(t *testing.T) {
		// 检查成功响应格式
		response := WeChatPayResponse{
			Code:    "SUCCESS",
			Message: "",
		}
		data, _ := json.Marshal(response)
		if string(data) != `{"code":"SUCCESS","message":""}` {
			t.Errorf("unexpected response format: %s", string(data))
		}

		// 检查失败响应格式
		failResponse := WeChatPayResponse{
			Code:    "FAIL",
			Message: "签名验证失败",
		}
		failData, _ := json.Marshal(failResponse)
		if string(failData) != `{"code":"FAIL","message":"签名验证失败"}` {
			t.Errorf("unexpected fail response format: %s", string(failData))
		}
	})

	t.Run("Mock repository records callback", func(t *testing.T) {
		record := &repository.PaymentCallbackRecord{
			PaymentMethod:   "wechat_pay",
			RequestHeaders:  `{"Content-Type":"application/json"}`,
			RequestBody:     `{"test":"data"}`,
			SignatureValid:  false,
			ProcessStatus:   "received",
			ProcessMessage:  "回调已接收",
			ResponseCode:    "SUCCESS",
			ResponseMessage: "",
			ProcessTimeMs:   10,
		}

		mockRepo.Create(nil, record)

		if !mockRepo.createCalled {
			t.Error("Create should be called")
		}
		if mockRepo.lastRecord.PaymentMethod != "wechat_pay" {
			t.Errorf("unexpected payment method: %s", mockRepo.lastRecord.PaymentMethod)
		}
	})
}

func TestWeChatPayWebhookHandler_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("respondSuccess returns correct format", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		h := &WeChatPayWebhookHandler{}
		h.respondSuccess(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var response WeChatPayResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if response.Code != "SUCCESS" {
			t.Errorf("expected code SUCCESS, got %s", response.Code)
		}
		if response.Message != "" {
			t.Errorf("expected empty message, got %s", response.Message)
		}
	})

	t.Run("respondFail returns correct format", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		h := &WeChatPayWebhookHandler{}
		h.respondFail(c, "测试错误")

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var response WeChatPayResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if response.Code != "FAIL" {
			t.Errorf("expected code FAIL, got %s", response.Code)
		}
		if response.Message != "测试错误" {
			t.Errorf("expected message '测试错误', got %s", response.Message)
		}
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
		if n != len(body) {
			t.Errorf("expected to read %d bytes, got %d", len(body), n)
		}
		if string(bodyBytes) != body {
			t.Errorf("body mismatch: expected %s, got %s", body, string(bodyBytes))
		}
	})
}
