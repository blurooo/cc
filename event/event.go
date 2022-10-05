// Package event 提供简单的事件发布订阅机制
package event

// Handler 订阅处理函数
type Handler func(conditions, payload map[string]interface{}) error

// Emitter 事件类型
type Emitter interface {
	// Sub 订阅某个事件，触发时将回调handler方法
	// 其中conditions是流水线on携带的条件，handler将负责这些条件的匹配，以及最终触发的所有逻辑
	Sub(eventType string, conditions map[string]interface{}, handler Handler)
	// Pub 发布某个事件，事件将携带payload
	Pub(eventType string, payload map[string]interface{}) error
}

// subscriber 事件订阅的具体信息
type subscriber struct {
	Conditions map[string]interface{}
	Handler    Handler
}

// Emit 事件触发器
type Emit struct {
	Subscribers map[string][]subscriber
}

// NewEmitter 实例化事件触发器
func NewEmitter() Emitter {
	return &Emit{
		Subscribers: map[string][]subscriber{},
	}
}

// Sub 实现发布，同个事件可以有多个订阅者
func (s *Emit) Sub(eventType string, conditions map[string]interface{}, handler Handler) {
	s.Subscribers[eventType] = append(s.Subscribers[eventType], subscriber{
		Conditions: conditions,
		Handler:    handler,
	})
}

// Pub 实现发布
func (s *Emit) Pub(eventType string, payload map[string]interface{}) error {
	subscribers, ok := s.Subscribers[eventType]
	if !ok {
		return nil
	}
	// 调用函数
	for _, subscriber := range subscribers {
		err := subscriber.Handler(subscriber.Conditions, payload)
		if err != nil {
			return err
		}
	}
	return nil
}
