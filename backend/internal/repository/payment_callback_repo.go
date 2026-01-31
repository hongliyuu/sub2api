package repository

import (
	"context"
	"log"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentcallback"
)

// PaymentCallbackRepository 支付回调记录仓库
type PaymentCallbackRepository struct {
	client *ent.Client
}

// NewPaymentCallbackRepository 创建支付回调记录仓库
func NewPaymentCallbackRepository(client *ent.Client) *PaymentCallbackRepository {
	return &PaymentCallbackRepository{client: client}
}

// PaymentCallbackRecord 支付回调记录
type PaymentCallbackRecord struct {
	OrderNo         *string // 订单号（从回调数据解析）
	PaymentMethod   string  // 支付方式
	TransactionID   *string // 支付平台订单号
	RequestHeaders  string  // 请求头 JSON
	RequestBody     string  // 请求体
	SignatureValid  bool    // 签名验证结果
	ProcessStatus   string  // 处理状态
	ProcessMessage  string  // 处理结果描述
	ResponseCode    string  // 响应码
	ResponseMessage string  // 响应消息
	ProcessTimeMs   int64   // 处理耗时（毫秒）
}

// Create 创建支付回调记录
func (r *PaymentCallbackRepository) Create(ctx context.Context, record *PaymentCallbackRecord) (*ent.PaymentCallback, error) {
	builder := r.client.PaymentCallback.Create().
		SetPaymentMethod(record.PaymentMethod).
		SetRequestHeaders(record.RequestHeaders).
		SetRequestBody(record.RequestBody).
		SetSignatureValid(record.SignatureValid).
		SetProcessStatus(record.ProcessStatus).
		SetProcessMessage(record.ProcessMessage).
		SetResponseCode(record.ResponseCode).
		SetResponseMessage(record.ResponseMessage).
		SetProcessTimeMs(record.ProcessTimeMs)

	if record.OrderNo != nil {
		builder.SetOrderNo(*record.OrderNo)
	}
	if record.TransactionID != nil {
		builder.SetTransactionID(*record.TransactionID)
	}

	callback, err := builder.Save(ctx)
	if err != nil {
		log.Printf("[PaymentCallbackRepo] Create failed: %v", err)
		return nil, err
	}

	return callback, nil
}

// Update 更新支付回调记录
func (r *PaymentCallbackRepository) Update(ctx context.Context, id int64, updates *PaymentCallbackRecord) (*ent.PaymentCallback, error) {
	builder := r.client.PaymentCallback.UpdateOneID(id)

	if updates.OrderNo != nil {
		builder.SetOrderNo(*updates.OrderNo)
	}
	if updates.TransactionID != nil {
		builder.SetTransactionID(*updates.TransactionID)
	}
	builder.SetSignatureValid(updates.SignatureValid)
	builder.SetProcessStatus(updates.ProcessStatus)
	builder.SetProcessMessage(updates.ProcessMessage)
	builder.SetResponseCode(updates.ResponseCode)
	builder.SetResponseMessage(updates.ResponseMessage)
	builder.SetProcessTimeMs(updates.ProcessTimeMs)

	callback, err := builder.Save(ctx)
	if err != nil {
		log.Printf("[PaymentCallbackRepo] Update failed: id=%d, error=%v", id, err)
		return nil, err
	}

	return callback, nil
}

// GetByID 根据 ID 获取记录
func (r *PaymentCallbackRepository) GetByID(ctx context.Context, id int64) (*ent.PaymentCallback, error) {
	return r.client.PaymentCallback.Get(ctx, id)
}

// ListByOrderNo 根据订单号查询回调记录
func (r *PaymentCallbackRepository) ListByOrderNo(ctx context.Context, orderNo string) ([]*ent.PaymentCallback, error) {
	return r.client.PaymentCallback.Query().
		Where(paymentcallback.OrderNoEQ(orderNo)).
		Order(ent.Desc(paymentcallback.FieldCreatedAt)).
		All(ctx)
}
