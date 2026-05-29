package runstate

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rotemmiz/forge/internal/engine/message"
)

func TestEnsureRunning_RunsWork(t *testing.T) {
	rs := New()
	want := message.WithParts{Info: message.Info{User: &message.UserMessage{ID: "msg_1"}}}
	got, err := rs.EnsureRunning(context.Background(), "ses_1", func(context.Context) (message.WithParts, error) {
		return want, nil
	})
	if err != nil || got.Info.ID() != "msg_1" {
		t.Fatalf("got %+v err %v", got, err)
	}
	if rs.Busy("ses_1") {
		t.Fatalf("session should be idle after completion")
	}
}

func TestEnsureRunning_Idempotent(t *testing.T) {
	rs := New()
	var runs int32
	release := make(chan struct{})
	work := func(context.Context) (message.WithParts, error) {
		atomic.AddInt32(&runs, 1)
		<-release
		return message.WithParts{Info: message.Info{User: &message.UserMessage{ID: "msg_1"}}}, nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = rs.EnsureRunning(context.Background(), "ses_1", work)
		}()
	}
	// Give the goroutines time to coalesce onto one run.
	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()

	if n := atomic.LoadInt32(&runs); n != 1 {
		t.Fatalf("work ran %d times, want 1 (idempotent)", n)
	}
}

func TestCancel_InterruptsRun(t *testing.T) {
	rs := New()
	started := make(chan struct{})
	work := func(ctx context.Context) (message.WithParts, error) {
		close(started)
		<-ctx.Done()
		return message.WithParts{}, ctx.Err()
	}
	done := make(chan error, 1)
	go func() {
		_, err := rs.EnsureRunning(context.Background(), "ses_1", work)
		done <- err
	}()
	<-started
	rs.Cancel("ses_1")
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("want context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancel did not interrupt the run")
	}
}

func TestAssertNotBusy(t *testing.T) {
	rs := New()
	started := make(chan struct{})
	release := make(chan struct{})
	go func() {
		_, _ = rs.EnsureRunning(context.Background(), "ses_1", func(context.Context) (message.WithParts, error) {
			close(started)
			<-release
			return message.WithParts{}, nil
		})
	}()
	<-started
	var busyErr *BusyError
	if err := rs.AssertNotBusy("ses_1"); !errors.As(err, &busyErr) {
		t.Fatalf("want BusyError, got %v", err)
	}
	if err := rs.AssertNotBusy("ses_other"); err != nil {
		t.Fatalf("idle session should not be busy: %v", err)
	}
	close(release)
}
