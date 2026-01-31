package webhook

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/gin-gonic/gin"
)

// 请求体最大大小限制（1MB）
const maxRequestBodySize = 1 << 20

// WeChatPayWebhookHandler 微信支付回调处理器
type WeChatPayWebhookHandler struct {
	callbackRepo *repository.PaymentCallbackRepository
}

// NewWeChatPayWebhookHandler 创建微信支付回调处理器
func NewWeChatPayWebhookHandler(callbackRepo *repository.PaymentCallbackRepository) *WeChatPayWebhookHandler {
	return &WeChatPayWebhookHandler{
		callbackRepo: callbackRepo,
	}
}

// WeChatPayResponse 微信支付回调响应格式
type WeChatPayResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// HandlePaymentNotify 处理微信支付回调通知
// POST /api/v1/webhook/wechat/payment
func (h *WeChatPayWebhookHandler) HandlePaymentNotify(c *gin.Context) {
	startTime := time.Now()

	// 读取请求体（限制大小防止内存耗尽攻击）
	bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, maxRequestBodySize))
	if err != nil {
		log.Printf("[WeChatPayWebhook] Failed to read request body: %v", err)
		h.respondFail(c, "读取请求体失败")
		return
	}
	bodyStr := string(bodyBytes)

	// 提取必要的请求头（仅记录微信支付相关头部）
	headers := extractWeChatPayHeaders(c.Request.Header)
	headersJSON, _ := json.Marshal(headers)

	log.Printf("[WeChatPayWebhook] Received callback: body_length=%d", len(bodyStr))

	// 创建回调记录
	record := &repository.PaymentCallbackRecord{
		PaymentMethod:   "wechat_pay",
		RequestHeaders:  string(headersJSON),
		RequestBody:     bodyStr,
		SignatureValid:  false, // 后续 Story 3-2 实现签名验证
		ProcessStatus:   "received",
		ProcessMessage:  "回调已接收，待处理",
		ResponseCode:    "",
		ResponseMessage: "",
		ProcessTimeMs:   0,
	}

	callback, err := h.callbackRepo.Create(c.Request.Context(), record)
	if err != nil {
		log.Printf("[WeChatPayWebhook] Failed to save callback record: %v", err)
		// 即使记录保存失败，也返回 SUCCESS 以避免微信重复发送
		h.respondSuccess(c)
		return
	}

	// 计算处理耗时
	processTimeMs := time.Since(startTime).Milliseconds()

	// 更新记录：标记为接收成功
	record.ProcessStatus = "received"
	record.ProcessMessage = "回调已接收，签名验证待后续 Story 实现"
	record.ResponseCode = "SUCCESS"
	record.ResponseMessage = ""
	record.ProcessTimeMs = processTimeMs

	_, updateErr := h.callbackRepo.Update(c.Request.Context(), callback.ID, record)
	if updateErr != nil {
		log.Printf("[WeChatPayWebhook] Failed to update callback record: %v", updateErr)
	}

	log.Printf("[WeChatPayWebhook] Callback processed: id=%d, process_time=%dms", callback.ID, processTimeMs)

	// 返回成功响应
	h.respondSuccess(c)
}

// respondSuccess 返回成功响应
func (h *WeChatPayWebhookHandler) respondSuccess(c *gin.Context) {
	c.JSON(http.StatusOK, WeChatPayResponse{
		Code:    "SUCCESS",
		Message: "",
	})
}

// respondFail 返回失败响应
func (h *WeChatPayWebhookHandler) respondFail(c *gin.Context, message string) {
	c.JSON(http.StatusOK, WeChatPayResponse{
		Code:    "FAIL",
		Message: message,
	})
}

// extractWeChatPayHeaders 提取微信支付相关的请求头
// 只记录签名验证所需的头部，避免记录敏感信息
func extractWeChatPayHeaders(header http.Header) map[string]string {
	headers := make(map[string]string)

	// 微信支付签名验证所需的头部
	wechatPayHeaders := []string{
		"Wechatpay-Timestamp",
		"Wechatpay-Nonce",
		"Wechatpay-Signature",
		"Wechatpay-Signature-Type",
		"Wechatpay-Serial",
		"Content-Type",
	}

	for _, key := range wechatPayHeaders {
		if values := header.Get(key); values != "" {
			headers[key] = values
		}
	}

	return headers
}
