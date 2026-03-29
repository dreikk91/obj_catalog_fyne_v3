package eventbus

import "testing"

func TestBus_PublishSubscribeAndUnsubscribe(t *testing.T) {
	bus := NewBus()
	called := 0
	unsubscribe := bus.Subscribe(TopicObjectSaved, func(payload any) {
		event, ok := payload.(ObjectSavedEvent)
		if !ok {
			t.Fatalf("unexpected payload type: %T", payload)
		}
		if event.ObjectID != 42 {
			t.Fatalf("unexpected object id: %d", event.ObjectID)
		}
		called++
	})

	bus.Publish(TopicObjectSaved, ObjectSavedEvent{ObjectID: 42})
	if called != 1 {
		t.Fatalf("expected handler call once, got %d", called)
	}

	unsubscribe()
	bus.Publish(TopicObjectSaved, ObjectSavedEvent{ObjectID: 42})
	if called != 1 {
		t.Fatalf("handler must not be called after unsubscribe, got %d", called)
	}
}

func TestBus_PublishNoSubscribers(t *testing.T) {
	bus := NewBus()
	bus.Publish(TopicDataRefresh, DataRefreshEvent{RefreshObjects: true})
}
