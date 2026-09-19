package micropython

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestMaintenanceFeedsWatchdogOnEveryCommandAndHardResets(t *testing.T) {
	port := newScriptedPort([]byte(">R\x00OK\x04\x04>R\x00OKone\x04\x04>R\x00OKtwo\x04\x04>>> __MICROS_RESET_READY__\r\n>>> "))
	client := newClient(port, time.Second)
	if err := client.startMaintenance(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"print('one')", "print('two')"} {
		if _, err := client.Exec(context.Background(), code); err != nil {
			t.Fatal(err)
		}
	}
	writes := port.writes.String()
	if !bytes.Contains([]byte(writes), []byte("machine.WDT(timeout=60000)")) || bytes.Count([]byte(writes), []byte("_micros_wdt.feed()")) != 3 {
		t.Fatalf("watchdog not maintained: %q", writes)
	}
	if err := client.Reset(); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasSuffix(port.writes.Bytes(), []byte("machine.reset()\r")) || client.watchdog {
		t.Fatal("maintenance must finish with a hard reset")
	}
}

func TestFilesystemInventoryKeepsHiddenFilesAndRejectsTraversal(t *testing.T) {
	entries, err := parseEntries([]byte("__MICROS_ENTRY__[\".guimeta.key\",32768,10]\r\n__MICROS_ENTRY__[\"empty\",16384,99]\r\n"), "/config")
	if err != nil || len(entries) != 2 || entries[0].Path != "/config/.guimeta.key" || !entries[1].Directory || entries[1].Size != 0 {
		t.Fatalf("bad inventory: %v %v", entries, err)
	}
	for _, row := range []string{`["../escape",32768,10]`, `["link",40960,10]`, `["",32768,10]`, `["negative",32768,-1]`} {
		if _, err := parseEntries([]byte("__MICROS_ENTRY__"+row), "/config"); err == nil {
			t.Errorf("accepted %s", row)
		}
	}
}

func TestReadFileRejectsChecksumMismatch(t *testing.T) {
	port := newScriptedPort([]byte(">R\x00OK__MICROS_FILE__616263\r\n\x04\x04>R\x00OK__MICROS_SHA256__0000000000000000000000000000000000000000000000000000000000000000\r\n\x04\x04"))
	_, err := newClient(port, time.Second).ReadFile(context.Background(), "/config/.guimeta.key")
	if err == nil {
		t.Fatal("corrupted download accepted for backup")
	}
}

type bootingPort struct {
	*scriptedPort
	attempts int
}

func (p *bootingPort) Read(data []byte) (int, error) {
	if p.input.Len() == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	return p.scriptedPort.Read(data)
}

func (p *bootingPort) Write(data []byte) (int, error) {
	if bytes.Equal(data, []byte{'\r', 0x01}) {
		p.attempts++
		if p.attempts == 2 {
			p.input = bytes.NewReader([]byte("raw REPL; CTRL-B to exit\r\n>"))
		}
	}
	if bytes.Equal(data, []byte{0x04}) {
		p.input = bytes.NewReader([]byte("soft reboot\r\nraw REPL; CTRL-B to exit\r\n>"))
	}
	return p.scriptedPort.Write(data)
}

func TestEnterRawREPLRetriesInterruptLostDuringBoot(t *testing.T) {
	port := &bootingPort{scriptedPort: newScriptedPort(nil)}
	if err := newClient(port, 3*time.Second).enterRawREPL(context.Background()); err != nil {
		t.Fatal(err)
	}
	if port.attempts != 2 {
		t.Fatalf("got %d entry attempts", port.attempts)
	}
}

type noDrainPort struct{ *scriptedPort }

func (p noDrainPort) Drain() error { panic("reset must not call an unbounded driver drain") }

func TestResetWaitsForInterpreterAcknowledgement(t *testing.T) {
	ready := ">>> __MICROS_RESET_READY__\r\n>>> "
	for _, response := range []string{"", ">>> ", ">>> __MICROS_RESET_READY__\r\n", ready} {
		port := noDrainPort{newScriptedPort([]byte(response))}
		err := newClient(port, 200*time.Millisecond).Reset()
		if (err == nil) != (response == ready) {
			t.Fatalf("response %q: %v", response, err)
		}
		if response != ready && bytes.Contains(port.writes.Bytes(), []byte("machine.reset")) {
			t.Fatal("sent reset before interpreter was ready")
		}
	}
}
