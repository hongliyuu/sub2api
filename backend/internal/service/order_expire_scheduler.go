package service

import (
	"context"
	"log"
	"sync"
	"time"
)

// OrderExpireScheduler 订单过期处理调度器
// 定时扫描过期的 pending 订单并将其标记为 expired
type OrderExpireScheduler struct {
	repo   RechargeOrderRepository
	stopCh chan struct{}
	wg     sync.WaitGroup

	// 配置
	interval int // 执行间隔（秒）
	limit    int // 每次最多处理的订单数
}

// NewOrderExpireScheduler 创建订单过期调度器
func NewOrderExpireScheduler(repo RechargeOrderRepository) *OrderExpireScheduler {
	return &OrderExpireScheduler{
		repo:     repo,
		stopCh:   make(chan struct{}),
		interval: 60,  // 默认每分钟执行一次
		limit:    100, // 默认每次最多处理 100 条
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

	count, err := s.repo.MarkExpiredOrders(ctx, s.limit)
	if err != nil {
		log.Printf("[OrderExpireScheduler] Error marking expired orders: %v", err)
		return
	}

	if count > 0 {
		log.Printf("[OrderExpireScheduler] Marked %d orders as expired", count)
	}
}
