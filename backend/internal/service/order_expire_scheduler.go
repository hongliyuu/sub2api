package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// OrderExpireScheduler 订单过期处理调度器
// 定时扫描过期的 pending 订单并将其标记为 expired
// 同时异步调用微信关闭订单 API
type OrderExpireScheduler struct {
	repo             RechargeOrderRepository
	wechatPayService *WeChatPayService
	stopCh           chan struct{}
	wg               sync.WaitGroup

	// 配置
	interval int // 执行间隔（秒）
	limit    int // 每次最多处理的订单数
}

// NewOrderExpireScheduler 创建订单过期调度器
func NewOrderExpireScheduler(repo RechargeOrderRepository, wechatPayService *WeChatPayService) *OrderExpireScheduler {
	return &OrderExpireScheduler{
		repo:             repo,
		wechatPayService: wechatPayService,
		stopCh:           make(chan struct{}),
		interval:         60,  // 默认每分钟执行一次
		limit:            100, // 默认每次最多处理 100 条
	}
}

// Start 启动调度器
func (s *OrderExpireScheduler) Start() {
	s.wg.Add(1)
	go s.run()
	log.Printf("[OrderExpireScheduler] Started (interval: %ds, limit: %d)", s.interval, s.limit)
}

// Stop 停止调度器
func (s *OrderExpireScheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	log.Printf("[OrderExpireScheduler] Stopped")
}

func (s *OrderExpireScheduler) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Duration(s.interval) * time.Second)
	defer ticker.Stop()

	// 启动时立即执行一次
	s.processExpiredOrders()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.processExpiredOrders()
		}
	}
}

// processExpiredOrders 处理过期订单
func (s *OrderExpireScheduler) processExpiredOrders() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	orderNos, err := s.repo.MarkExpiredOrders(ctx, s.limit)
	if err != nil {
		log.Printf("[OrderExpireScheduler] Error marking expired orders: %v", err)
		return
	}

	if len(orderNos) > 0 {
		log.Printf("[OrderExpireScheduler] Marked %d orders as expired", len(orderNos))

		// 异步调用微信关闭订单 API
		for _, orderNo := range orderNos {
			go s.closeWeChatOrder(orderNo)
		}
	}
}

// closeWeChatOrder 异步关闭微信支付订单
// 失败时只记录日志，不阻塞主流程
func (s *OrderExpireScheduler) closeWeChatOrder(orderNo string) {
	// 检查微信支付服务是否可用
	if s.wechatPayService == nil || !s.wechatPayService.IsEnabled() {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := s.wechatPayService.CloseOrder(ctx, orderNo)
	if err != nil {
		log.Printf("[OrderExpireScheduler] Failed to close WeChat order: order_no=%s, error=%v", orderNo, err)
		// 记录关闭失败到订单备注
		s.appendOrderNotes(orderNo, fmt.Sprintf("微信关单失败: %v", err))
		return
	}

	// 记录关闭成功到订单备注
	s.appendOrderNotes(orderNo, "微信关单成功")
}

// appendOrderNotes 追加订单备注（异步，失败不阻塞）
func (s *OrderExpireScheduler) appendOrderNotes(orderNo, notes string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.repo.AppendNotes(ctx, orderNo, notes); err != nil {
		log.Printf("[OrderExpireScheduler] Failed to append notes: order_no=%s, error=%v", orderNo, err)
	}
}
