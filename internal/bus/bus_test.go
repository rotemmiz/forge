package bus

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewEventPropertiesNeverNil(t *testing.T) {
	e := NewEvent(EventConnected, nil)
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	// properties must serialize as {} (not null) to match opencode.
	if got := string(b); !strings.Contains(got, `"properties":{}`) {
		t.Errorf("marshaled event = %s, want properties:{}", got)
	}
	if e.ID == "" || e.Type != EventConnected {
		t.Errorf("bad event: %+v", e)
	}
}

func TestInstancePublishFanOut(t *testing.T) {
	b := NewInstanceBus("/dir", nil)
	ch, unsub := b.Subscribe()
	defer unsub()

	b.Publish(NewEvent("x.test", nil))
	select {
	case e := <-ch:
		if e.Type != "x.test" {
			t.Errorf("got %q", e.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive published event")
	}
}

func TestUnsubscribeClosesChannel(t *testing.T) {
	b := NewInstanceBus("/dir", nil)
	ch, unsub := b.Subscribe()
	unsub()
	if _, ok := <-ch; ok {
		t.Error("channel should be closed after unsubscribe")
	}
	// A publish after unsubscribe must not panic (no subscribers).
	b.Publish(NewEvent("x.test", nil))
}

func TestPublishForwardsToGlobalWithDirectory(t *testing.T) {
	g := NewGlobal()
	gch, gunsub := g.Subscribe()
	defer gunsub()

	b := NewInstanceBus("/proj", g)
	b.Publish(NewEvent("x.test", nil))

	select {
	case ge := <-gch:
		if ge.Directory != "/proj" {
			t.Errorf("global event directory = %q, want /proj", ge.Directory)
		}
		if ge.Payload.Type != "x.test" {
			t.Errorf("payload type = %q", ge.Payload.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("global bus did not receive forwarded event")
	}
}

func TestGlobalConnectedEnvelopeHasNoDirectory(t *testing.T) {
	// A server.connected injected directly into the global stream carries only
	// the payload (no directory), matching the recorded truth.
	ge := GlobalEvent{Payload: NewEvent(EventConnected, nil)}
	b, err := json.Marshal(ge)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); strings.Contains(got, `"directory"`) {
		t.Errorf("connected envelope must omit directory, got %s", got)
	}
}

func TestSlowSubscriberDoesNotBlock(t *testing.T) {
	b := NewInstanceBus("/dir", nil)
	_, unsub := b.Subscribe() // never drained
	defer unsub()
	done := make(chan struct{})
	go func() {
		for i := 0; i < subBuffer+10; i++ {
			b.Publish(NewEvent("x.flood", nil))
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("publish blocked on a slow subscriber")
	}
}
