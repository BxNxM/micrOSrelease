package micropython

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

type resetDisconnectPort struct {
	*scriptedPort
	disconnect      error
	resetError      error
	resetSent       bool
	readsAfterReset int
}

func (p *resetDisconnectPort) Read(data []byte) (int, error) {
	if p.resetSent {
		p.readsAfterReset++
		return 0, p.disconnect
	}
	if p.input.Len() == 0 {
		return 0, p.disconnect
	}
	return p.scriptedPort.Read(data)
}

func (p *resetDisconnectPort) Write(data []byte) (int, error) {
	if bytes.Equal(data, []byte("machine.reset()\r")) {
		if p.resetError != nil {
			return 0, p.resetError
		}
		p.resetSent = true
	}
	return p.scriptedPort.Write(data)
}

func (p *resetDisconnectPort) Drain() error { panic("reset must not drain a rebooting USB device") }

func TestResetHandlesUSBDisconnectOnlyAfterAcknowledgedPreparation(t *testing.T) {
	ready := ">>> __MICROS_RESET_READY__\r\n>>> "
	for _, tc := range []struct {
		name, response string
		writeFailure   bool
		wantSuccess    bool
	}{
		{name: "disconnect before prompt"},
		{name: "disconnect before acknowledgement", response: ">>> "},
		{name: "disconnect before ready prompt", response: ">>> __MICROS_RESET_READY__\r\n"},
		{name: "command echo is not acknowledgement", response: ">>> import machine, os; os.sync(); print('__MICROS_RESET_READY__')\r\n"},
		{name: "reset write fails", response: ready, writeFailure: true},
		{name: "USB disappears on reset", response: ready, wantSuccess: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			disconnected := errors.New("device not configured")
			port := &resetDisconnectPort{scriptedPort: newScriptedPort([]byte(tc.response)), disconnect: disconnected}
			if tc.writeFailure {
				port.resetError = disconnected
			}
			client := newClient(port, time.Second)
			client.inRaw, client.watchdog = true, true
			err := client.Reset()
			if (err == nil) != tc.wantSuccess || (!tc.wantSuccess && !errors.Is(err, disconnected)) {
				t.Fatalf("unexpected reset result: %v", err)
			}
			if port.resetSent != tc.wantSuccess || port.readsAfterReset != 0 || client.watchdog == tc.wantSuccess {
				t.Fatalf("unsafe reset state: sent=%v readsAfterReset=%d watchdog=%v", port.resetSent, port.readsAfterReset, client.watchdog)
			}
			if tc.wantSuccess {
				if !strings.Contains(port.writes.String(), "os.sync()") {
					t.Fatal("reset did not synchronize the filesystem")
				}
				beforeClose := port.writes.String()
				if err := client.Close(); err != nil || beforeClose != port.writes.String() {
					t.Fatal("closing sent REPL commands to the restarting device")
				}
			}
		})
	}
}
