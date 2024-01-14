/*
This package provides a portable intaface to pubsub model.
PubSub can publish/subscribe/unsubscribe messages for all.
To subscribe:

	ps := pubsub.New()
	ps.Sub(func(s string) {
		fmt.Println(s)
	})

To publish:

	ps.Pub("hello world")

The message are allowed to pass any types, and passing to subscribers which
can accept the type for the argument of callback.
*/
package pubsub

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sync"
)

type SubscriptionError struct {
	subscriber any
	err        any
}

func (pse *SubscriptionError) String() string {
	return fmt.Sprintf("%v: %v", pse.subscriber, pse.err)
}

func (pse *SubscriptionError) Error() string {
	return fmt.Sprint(pse.err)
}

func (pse *SubscriptionError) Subscriber() any {
	return pse.subscriber
}

type closure struct {
	function any
}

func Closure(function any) *closure {
	return &closure{function: function}
}

// Broker contains channel and callbacks.
type Broker struct {
	eventChannel chan any
	functions    []*closure
	mutex        sync.Mutex
	errorChannel chan error
}

// New return new PubSub intreface.
func New() *Broker {
	ps := new(Broker)
	ps.eventChannel = make(chan any)
	ps.errorChannel = make(chan error)
	call := func(f any, rf reflect.Value, in []reflect.Value) {
		defer func() {
			if err := recover(); err != nil {
				ps.errorChannel <- &SubscriptionError{f, err}
			}
		}()
		rf.Call(in)
	}

	go func() {
		for v := range ps.eventChannel {
			rv := reflect.ValueOf(v)
			ps.mutex.Lock()
			for _, w := range ps.functions {
				rf := reflect.ValueOf(w.function)
				if rv.Type() == reflect.ValueOf(w.function).Type().In(0) {
					go call(w.function, rf, []reflect.Value{rv})
				}
			}
			ps.mutex.Unlock()
		}
	}()
	return ps
}

func (ps *Broker) Error() chan error {
	return ps.errorChannel
}

// Subscribe subscribe to the PubSub.
func (ps *Broker) Subscribe(f any) error {
	check := f
	w, wrapped := f.(*closure)
	if wrapped {
		check = w.function
	}
	rf := reflect.ValueOf(check)
	if rf.Kind() != reflect.Func {
		return errors.New("not a function")
	}
	if rf.Type().NumIn() != 1 {
		return errors.New("number of arguments must be 1")
	}
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	if w, wrapped := f.(*closure); wrapped {
		ps.functions = append(ps.functions, w)
	} else {
		ps.functions = append(ps.functions, &closure{function: f})
	}
	return nil
}

// Unsubscribe unsubscribe to the PubSub.
func (ps *Broker) Unsubscribe(f any) {
	var fp uintptr
	if f == nil {
		if pc, _, _, ok := runtime.Caller(1); ok {
			fp = runtime.FuncForPC(pc).Entry()
		}
	} else {
		fp = reflect.ValueOf(f).Pointer()
	}
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	result := make([]*closure, 0, len(ps.functions))
	last := 0
	for i, v := range ps.functions {
		vf := v.function
		if reflect.ValueOf(vf).Pointer() == fp {
			result = append(result, ps.functions[last:i]...)
			last = i + 1
		}
	}
	ps.functions = append(result, ps.functions[last:]...)
}

// Publish publish to the PubSub.
func (ps *Broker) Publish(v any) {
	ps.eventChannel <- v
}

// Close closes PubSub. To inspect unbsubscribing for another subscruber, you must create message structure to notify them. After publish notifycations, Close should be called.
func (ps *Broker) Close() {
	close(ps.eventChannel)
	ps.functions = nil
}
