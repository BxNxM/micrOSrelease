package network

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestStatusWithFragmentedPromptAndAuthentication(t *testing.T) {
	for _, tc := range []struct {
		name, reply, want string
	}{
		{"auth enabled", "True", "ON"},
		{"auth disabled", "False", "OFF"},
		{"auth unavailable", "Unknown parameter", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			done := make(chan error, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					done <- err
					return
				}
				defer conn.Close()
				conn.SetDeadline(time.Now().Add(5 * time.Second))
				fmt.Fprint(conn, "[password] test ")
				fmt.Fprint(conn, "$ ")
				commands := []string{"secret", "hello", "version", "conf", "webui", "espnow", "auth", "cron", "timirq"}
				replies := []string{"AuthOk", "hello:test:uid123:rel", "3.6.0-0", "", "True", "False", tc.reply, "True", "False"}
				for i, expected := range commands {
					buf := make([]byte, len(expected))
					offset := 0
					for offset < len(buf) {
						n, e := conn.Read(buf[offset:])
						if e != nil {
							done <- e
							return
						}
						offset += n
					}
					if string(buf) != expected {
						done <- fmt.Errorf("got %q want %q", buf, expected)
						return
					}
					prefix := ""
					if i >= 3 {
						prefix = "[configure] "
					}
					fmt.Fprint(conn, replies[i]+"\n"+prefix+"test ")
					fmt.Fprint(conn, "$ ")
				}
				done <- nil
			}()
			node, valid := inspect(context.Background(), Device{Address: listener.Addr().String()}, "secret")
			if !valid || !node.Online || node.UID != "uid123" || node.Version != "3.6.0-0" || node.Features["webui"] != "ON" || node.Features["espnow"] != "OFF" || node.Features["auth"] != tc.want || node.Features["cron"] != "ON" || node.Features["timirq"] != "OFF" {
				t.Fatalf("unexpected node: %+v", node)
			}
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestClientRejectsDisconnectBeforePrompt(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	go func() { fmt.Fprint(b, "partial reply"); b.Close() }()
	_, err := (&Client{conn: a}).receive(context.Background())
	if err == nil || !strings.Contains(err.Error(), "receive prompt") {
		t.Fatalf("expected framing error, got %v", err)
	}
}

func TestScanRangeValidation(t *testing.T) {
	for _, value := range []string{"10.0.0.0/8", "::1/128", "invalid"} {
		if _, err := scanPrefixes(value); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
	if _, err := scanPrefixes("127.0.0.1/32"); err != nil {
		t.Fatal(err)
	}
}
