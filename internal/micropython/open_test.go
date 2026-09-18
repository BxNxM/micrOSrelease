package micropython

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.bug.st/serial"
)

type closingPort struct {
	*scriptedPort
	done chan struct{}
	once sync.Once
}

func (p *closingPort) Close() error { p.once.Do(func() { close(p.done) }); return nil }

func TestBlockedSerialOpenCanBeCancelledWithoutLateWritesOrOverlappingOpens(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	port := &closingPort{scriptedPort: newScriptedPort(nil), done: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := openClient(ctx, "/dev/tty.test-cancel", 115200, time.Second, false, func(string, *serial.Mode) (serialPort, error) { close(entered); <-release; return port, nil })
		result <- err
	}()
	<-entered
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("blocked driver froze caller")
	}
	_, err := openClient(context.Background(), "/dev/cu.test-cancel", 115200, 20*time.Millisecond, false, func(string, *serial.Mode) (serialPort, error) {
		t.Error("overlapping native opens")
		return nil, errors.New("should not dial")
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("gate did not respect cancellation: %v", err)
	}
	close(release)
	select {
	case <-port.done:
	case <-time.After(time.Second):
		t.Fatal("late serial handle was not closed")
	}
	if port.writes.Len() != 0 {
		t.Fatal("abandoned open sent commands to reconnected board")
	}
	want := errors.New("new attempt")
	_, err = openClient(context.Background(), "/dev/cu.test-cancel", 115200, time.Second, false, func(string, *serial.Mode) (serialPort, error) { return nil, want })
	if !errors.Is(err, want) {
		t.Fatalf("new attempt blocked after cleanup: %v", err)
	}
}
