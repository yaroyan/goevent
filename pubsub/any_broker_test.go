package pubsub_test

import (
	"testing"
	"time"

	"github.com/yaroyan/goevent/pubsub"
)

type TestService struct {
}

type TestEvent struct {
	message string
	isRead  bool
}

func (t *TestService) getMessage(event *TestEvent) {
	event.isRead = true
}

type TestService2 struct {
}

type TestEvent2 struct {
	message string
	isRead  bool
}

func (t *TestService2) getMessage2(event *TestEvent2) {
	event.isRead = true
}

var service = new(TestService)
var service2 = new(TestService2)
var subscription = pubsub.NewSubscription(service.getMessage)
var subscription2 = pubsub.NewSubscription(service2.getMessage2)

func TestSubscribe(t *testing.T) {
	var broker = pubsub.NewEventBroker()
	if err := broker.Subscribe(subscription); err != nil {
		t.Error(err)
	}
	if !broker.IsSubscribing(subscription) {
		t.Error("have not the subscription")
	}
}

func TestUnsubscribe(t *testing.T) {
	var broker = pubsub.NewEventBroker()
	if err := broker.Subscribe(subscription); err != nil {
		t.Error(err)
	}
	if err := broker.Subscribe(subscription2); err != nil {
		t.Error(err)
	}

	broker.Unsubscribe(subscription)

	if broker.IsSubscribing(subscription) {
		t.Error("have the subscription")
	}
	if !broker.IsSubscribing(subscription2) {
		t.Error("have not the subscription2")
	}
}

func TestPublish(t *testing.T) {
	var broker = pubsub.NewEventBroker()
	if err := broker.Subscribe(subscription); err != nil {
		t.Error(err)
	}
	if err := broker.Subscribe(subscription2); err != nil {
		t.Error(err)
	}
	event := &TestEvent{message: "test event 1", isRead: false}
	broker.Publish(event)
	event2 := &TestEvent2{message: "test event 2", isRead: false}
	broker.Publish(event2)

	time.Sleep(100 * time.Millisecond)

	if !event.isRead {
		t.Error("event1 is unread")
	}
	if !event2.isRead {
		t.Error("event2 is unread")
	}
}
