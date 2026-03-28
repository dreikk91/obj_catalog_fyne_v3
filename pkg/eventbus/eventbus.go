package eventbus

import "sync"

// Topic визначає канал доменних подій.
type Topic string

const (
	TopicObjectSaved   Topic = "object.saved"
	TopicObjectDeleted Topic = "object.deleted"
	TopicDataRefresh   Topic = "data.refresh"
)

// Handler обробляє payload події.
type Handler func(payload any)

// Bus реалізує просту in-process pub/sub шину подій.
type Bus struct {
	mu          sync.RWMutex
	nextID      uint64
	subscribers map[Topic]map[uint64]Handler
}

func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[Topic]map[uint64]Handler),
	}
}

// Subscribe додає обробник для topic і повертає функцію відписки.
func (b *Bus) Subscribe(topic Topic, handler Handler) func() {
	if b == nil || handler == nil {
		return func() {}
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	id := b.nextID

	if _, ok := b.subscribers[topic]; !ok {
		b.subscribers[topic] = make(map[uint64]Handler)
	}
	b.subscribers[topic][id] = handler

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		handlers, ok := b.subscribers[topic]
		if !ok {
			return
		}
		delete(handlers, id)
		if len(handlers) == 0 {
			delete(b.subscribers, topic)
		}
	}
}

// Publish публікує payload в topic і синхронно викликає всі підписані обробники.
func (b *Bus) Publish(topic Topic, payload any) {
	if b == nil {
		return
	}

	b.mu.RLock()
	handlersMap, ok := b.subscribers[topic]
	if !ok || len(handlersMap) == 0 {
		b.mu.RUnlock()
		return
	}
	handlers := make([]Handler, 0, len(handlersMap))
	for _, handler := range handlersMap {
		handlers = append(handlers, handler)
	}
	b.mu.RUnlock()

	for _, handler := range handlers {
		handler(payload)
	}
}
