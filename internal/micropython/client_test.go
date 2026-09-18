package micropython

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

type scriptedPort struct {
	input  *bytes.Reader
	writes bytes.Buffer
	closed bool
}

func newScriptedPort(input []byte) *scriptedPort { return &scriptedPort{input: bytes.NewReader(input)} }
func (port *scriptedPort) Read(data []byte) (int, error) {
	n, err := port.input.Read(data)
	if err == io.EOF {
		return 0, nil
	}
	return n, err
}
func (port *scriptedPort) Write(data []byte) (int, error)     { return port.writes.Write(data) }
func (port *scriptedPort) Close() error                       { port.closed = true; return nil }
func (port *scriptedPort) ResetInputBuffer() error            { return nil }
func (port *scriptedPort) SetReadTimeout(time.Duration) error { return nil }

func TestEnterRawREPL(t *testing.T) {
	input := []byte("raw REPL; CTRL-B to exit\r\n>soft reboot\r\nraw REPL; CTRL-B to exit\r\n")
	port := newScriptedPort(input)
	client := newClient(port, time.Second)
	if err := client.enterRawREPL(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !client.inRaw || !bytes.Contains(port.writes.Bytes(), []byte{'\r', 0x01}) {
		t.Fatalf("raw REPL was not entered: %q", port.writes.Bytes())
	}
}

func TestExecRawFallback(t *testing.T) {
	port := newScriptedPort([]byte(">R\x00OKhello\r\n\x04\x04"))
	client := newClient(port, time.Second)
	client.inRaw = true
	output, err := client.Exec(context.Background(), "print('hello')")
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != "hello\r\n" {
		t.Fatalf("unexpected output %q", output)
	}
}

func TestExecReportsRemoteError(t *testing.T) {
	port := newScriptedPort([]byte(">R\x00OK\x04Traceback: failed\r\n\x04"))
	client := newClient(port, time.Second)
	client.inRaw = true
	if _, err := client.Exec(context.Background(), "bad()"); err == nil {
		t.Fatal("expected remote execution error")
	}
}

func TestExecRawPaste(t *testing.T) {
	input := append([]byte{'>'}, 'R', 0x01, 4, 0, 0x01, 0x01, 0x04)
	input = append(input, []byte("done\r\n")...)
	input = append(input, 0x04, 0x04)
	port := newScriptedPort(input)
	client := newClient(port, time.Second)
	client.inRaw = true
	output, err := client.Exec(context.Background(), "12345678")
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != "done\r\n" {
		t.Fatalf("unexpected output %q", output)
	}
}

func TestTimeoutUsesEarlierOfClientAndParentDeadline(t *testing.T) {
	client := newClient(newScriptedPort(nil), time.Second)
	for _, parentTimeout := range []time.Duration{10 * time.Millisecond, 10 * time.Minute} {
		parent, cancelParent := context.WithTimeout(context.Background(), parentTimeout)
		started := time.Now()
		ctx, cancel := client.withTimeout(parent)
		deadline, ok := ctx.Deadline()
		if !ok || deadline.After(started.Add(min(parentTimeout, client.timeout)+10*time.Millisecond)) {
			t.Errorf("REPL timeout exceeds earlier deadline: %v", deadline.Sub(started))
		}
		cancel()
		cancelParent()
	}
}
