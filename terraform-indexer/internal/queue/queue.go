package queue

import (
	"context"
	"sync"

	"github.com/nkbud/terraform-dashboard/terraform-indexer/internal/model"
)

// Queue represents a generic queue interface
type Queue[T any] interface {
	Enqueue(ctx context.Context, item T) error
	Dequeue(ctx context.Context) (T, error)
	Size() int
	Close() error
}

// FileQueue is a queue for TerraformFile objects
type FileQueue interface {
	Queue[*model.TerraformFile]
}

// ObjectQueue is a queue for TerraformObject objects
type ObjectQueue interface {
	Queue[*model.TerraformObject]
}

// InMemoryQueue is a simple in-memory queue implementation
type InMemoryQueue[T any] struct {
	items []T
	mutex sync.RWMutex
	cond  *sync.Cond
}

// NewInMemoryQueue creates a new in-memory queue
func NewInMemoryQueue[T any]() *InMemoryQueue[T] {
	q := &InMemoryQueue[T]{
		items: make([]T, 0),
	}
	q.cond = sync.NewCond(&q.mutex)
	return q
}

// Enqueue adds an item to the queue
func (q *InMemoryQueue[T]) Enqueue(ctx context.Context, item T) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	
	q.items = append(q.items, item)
	q.cond.Signal()
	return nil
}

// Dequeue removes and returns an item from the queue
func (q *InMemoryQueue[T]) Dequeue(ctx context.Context) (T, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	
	for len(q.items) == 0 {
		select {
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err()
		default:
			q.cond.Wait()
		}
	}
	
	item := q.items[0]
	q.items = q.items[1:]
	return item, nil
}

// Size returns the current size of the queue
func (q *InMemoryQueue[T]) Size() int {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return len(q.items)
}

// Close closes the queue
func (q *InMemoryQueue[T]) Close() error {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.cond.Broadcast()
	return nil
}

// NewFileQueue creates a new file queue
func NewFileQueue() FileQueue {
	return NewInMemoryQueue[*model.TerraformFile]()
}

// NewObjectQueue creates a new object queue
func NewObjectQueue() ObjectQueue {
	return NewInMemoryQueue[*model.TerraformObject]()
}