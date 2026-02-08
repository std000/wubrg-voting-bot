package bot

import (
	"log"
	"sync"
)

// UpdateQueue управляет очередью задач на обновление сообщений голосований.
// Задачи дедуплицируются по pollID и обрабатываются последовательно в одном потоке.
type UpdateQueue struct {
	mu      sync.Mutex
	pending map[int64]struct{} // множество pollID, ожидающих обновления
	notify  chan struct{}      // сигнальный канал для пробуждения воркера
}

// NewUpdateQueue создает новую очередь обновлений
func NewUpdateQueue() *UpdateQueue {
	return &UpdateQueue{
		pending: make(map[int64]struct{}),
		notify:  make(chan struct{}, 1),
	}
}

// Schedule добавляет pollID в очередь на обновление.
// Если pollID уже в очереди, повторно не добавляется (дедупликация).
func (q *UpdateQueue) Schedule(pollID int64) {
	q.mu.Lock()
	q.pending[pollID] = struct{}{}
	q.mu.Unlock()

	// Неблокирующая отправка уведомления воркеру
	select {
	case q.notify <- struct{}{}:
	default:
		// Воркер уже уведомлён, новое уведомление не нужно
	}

	log.Printf("📨 [UpdateQueue] Задача на обновление голосования %d добавлена в очередь", pollID)
}

// drain забирает все ожидающие pollID и очищает очередь
func (q *UpdateQueue) drain() []int64 {
	q.mu.Lock()
	defer q.mu.Unlock()

	polls := make([]int64, 0, len(q.pending))
	for pollID := range q.pending {
		polls = append(polls, pollID)
	}
	q.pending = make(map[int64]struct{})
	return polls
}
