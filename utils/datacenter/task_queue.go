package datacenter

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

// TaskQueue 可靠任务队列：BRPOPLPUSH 入 processing → 处理完成 LREM ACK；崩溃后 Reclaim 回主队列。
type TaskQueue struct {
	redis         *redis.Redis
	queueKey      string
	processingKey string
}

func NewTaskQueue(rds *redis.Redis, queueKey, processingKey string) *TaskQueue {
	return &TaskQueue{redis: rds, queueKey: queueKey, processingKey: processingKey}
}

// EnqueueTask 将任务 ID 推入待处理队列（LPUSH，与 BRPOPLPUSH 配合 FIFO）。
func EnqueueTask(ctx context.Context, rds *redis.Redis, queueKey string, taskID uint64) error {
	if rds == nil {
		return fmt.Errorf("redis unavailable")
	}
	if taskID == 0 {
		return fmt.Errorf("task id is zero")
	}
	_, err := rds.LpushCtx(ctx, queueKey, fmt.Sprintf("%d", taskID))
	return err
}

// Reserve 阻塞取出任务并移入 processing 队列（原子 BRPOPLPUSH）。
func (q *TaskQueue) Reserve(ctx context.Context, block time.Duration) (uint64, error) {
	if q == nil || q.redis == nil {
		return 0, fmt.Errorf("task queue unavailable")
	}
	sec := int(block.Seconds())
	if sec < 1 {
		sec = 1
	}
	v, err := q.redis.DoCtx(ctx, "BRPOPLPUSH", q.queueKey, q.processingKey, sec)
	if err != nil {
		// 旧版 Redis 无 BRPOPLPUSH 时降级为非阻塞 RPOPLPUSH
		if strings.Contains(strings.ToLower(err.Error()), "unknown command") {
			return q.reserveOnce(ctx)
		}
		return 0, err
	}
	return parseTaskID(v)
}

func (q *TaskQueue) reserveOnce(ctx context.Context) (uint64, error) {
	v, err := q.redis.DoCtx(ctx, "RPOPLPUSH", q.queueKey, q.processingKey)
	if err != nil {
		return 0, err
	}
	id, err := parseTaskID(v)
	if err != nil || id == 0 {
		return 0, err
	}
	return id, nil
}

// Ack 从 processing 队列移除已处理任务。
func (q *TaskQueue) Ack(ctx context.Context, taskID uint64) error {
	if q == nil || q.redis == nil || taskID == 0 {
		return nil
	}
	_, err := q.redis.LremCtx(ctx, q.processingKey, 1, fmt.Sprintf("%d", taskID))
	return err
}

// Requeue 将任务从 processing 移回主队列（处理失败需重试时）。
func (q *TaskQueue) Requeue(ctx context.Context, taskID uint64) error {
	if q == nil || q.redis == nil || taskID == 0 {
		return nil
	}
	idStr := fmt.Sprintf("%d", taskID)
	if _, err := q.redis.LremCtx(ctx, q.processingKey, 1, idStr); err != nil {
		return err
	}
	_, err := q.redis.LpushCtx(ctx, q.queueKey, idStr)
	return err
}

// ReclaimAll 启动/定时回收：将 processing 中全部任务移回主队列（at-least-once）。
func (q *TaskQueue) ReclaimAll(ctx context.Context) (int, error) {
	if q == nil || q.redis == nil {
		return 0, nil
	}
	items, err := q.redis.LrangeCtx(ctx, q.processingKey, 0, -1)
	if err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return 0, nil
	}
	count := 0
	// 从尾部往前移，尽量保持原顺序
	for i := len(items) - 1; i >= 0; i-- {
		idStr := strings.TrimSpace(items[i])
		if idStr == "" {
			continue
		}
		if _, err := q.redis.LremCtx(ctx, q.processingKey, 1, idStr); err != nil {
			continue
		}
		if _, err := q.redis.LpushCtx(ctx, q.queueKey, idStr); err != nil {
			continue
		}
		count++
	}
	return count, nil
}

func parseTaskID(v any) (uint64, error) {
	if v == nil {
		return 0, nil
	}
	switch x := v.(type) {
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0, nil
		}
		return strconv.ParseUint(s, 10, 64)
	case []byte:
		return strconv.ParseUint(strings.TrimSpace(string(x)), 10, 64)
	case int64:
		if x <= 0 {
			return 0, nil
		}
		return uint64(x), nil
	case int:
		if x <= 0 {
			return 0, nil
		}
		return uint64(x), nil
	default:
		return strconv.ParseUint(strings.TrimSpace(fmt.Sprint(v)), 10, 64)
	}
}
