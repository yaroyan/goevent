package pubsub

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

type SubscriptionError struct {
	subscription any
	err          any
}

func (se *SubscriptionError) String() string {
	return fmt.Sprintf("%v: %v", se.subscription, se.err)
}

func (se *SubscriptionError) Error() string {
	return fmt.Sprint(se.err)
}

func (se *SubscriptionError) Subscription() any {
	return se.subscription
}

// Subscription is function wrapper for Subscription.
type Subscription struct {
	function any
}

// NewSubscription creates new closure instance.
func NewSubscription[T any](function func(T)) *Subscription {
	return &Subscription{function: function}
}

// EventBroker contains channel and callbacks.
type EventBroker struct {
	eventChannel  chan any
	subscriptions map[reflect.Type][]*Subscription
	mutex         sync.Mutex
	errorChannel  chan error
}

func (eb *EventBroker) Error() chan error {
	return eb.errorChannel
}

// NewEventBroker return new Broker intreface.
func NewEventBroker() *EventBroker {
	eb := new(EventBroker)
	eb.subscriptions = make(map[reflect.Type][]*Subscription)
	eb.eventChannel = make(chan any)
	eb.errorChannel = make(chan error)
	call := func(f any, rf reflect.Value, in []reflect.Value) {
		defer func() {
			if err := recover(); err != nil {
				eb.errorChannel <- &SubscriptionError{f, err}
			}
		}()
		rf.Call(in)
	}

	go func() {
		for v := range eb.eventChannel {
			rv := reflect.ValueOf(v)
			eb.mutex.Lock()
			if s, e := eb.subscriptions[rv.Type()]; e {
				for _, w := range s {
					go call(w.function, reflect.ValueOf(w.function), []reflect.Value{rv})
				}
			}
			eb.mutex.Unlock()
		}
	}()
	return eb
}

// Subscribe subscribe to the Broker.
func (eb *EventBroker) Subscribe(subscription *Subscription) error {
	rf := reflect.ValueOf(subscription.function)
	if rf.Kind() != reflect.Func {
		return errors.New("not a function")
	}
	if rf.Type().NumIn() != 1 {
		return errors.New("number of arguments must be 1")
	}
	eb.mutex.Lock()
	defer eb.mutex.Unlock()
	argType := rf.Type().In(0)
	if _, exists := eb.subscriptions[argType]; !exists {
		eb.subscriptions[argType] = make([]*Subscription, 0)
	}
	eb.subscriptions[argType] = append(eb.subscriptions[argType], subscription)
	return nil
}

// Unsubscribe unsubscribe to the Broker.
func (eb *EventBroker) Unsubscribe(subscription *Subscription) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()
	subscriptions := make([]*Subscription, 0, len(eb.subscriptions))
	for k, v := range eb.subscriptions {
		last := 0
		for i, e := range v {
			if e == subscription {
				subscriptions = append(subscriptions, v[last:i]...)
				last = i + 1
			}
		}
		eb.subscriptions[k] = append(subscriptions, v[last:]...)
	}
}

// IsSubscribing check the subscription is active.
func (eb *EventBroker) IsSubscribing(subscription *Subscription) bool {
	for _, v := range eb.subscriptions {
		for _, e := range v {
			if e == subscription {
				return true
			}
		}
	}
	return false
}

// Publish publish to the Broker.
func (eb *EventBroker) Publish(v any) {
	eb.eventChannel <- v
}

// Close closes Broker.
// To inspect unbsubscribing for another subscruber, you must create message structure to notify them.
// After publish notifications, Close should be called.
func (eb *EventBroker) Close() {
	close(eb.eventChannel)
	eb.subscriptions = nil
}
