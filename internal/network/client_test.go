package network

import (
	"context"
	"fmt"
	"io"
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

func TestPromptPrefixesAndTrailingNewline(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	go func() {
		defer b.Close()
		for _, part := range []string{"value mentions node $ inside\n\x1b[1m[conf", "igure] node", " $ \x1b[0m\n"} {
			fmt.Fprint(b, part)
		}
	}()
	c := &Client{conn: a, prompt: "node"}
	reply, err := c.receive(context.Background())
	if err != nil || c.Prompt() != "[configure] node $ " {
		t.Fatalf("%q %v", c.Prompt(), err)
	}
	loc := promptPattern.FindStringIndex(reply)
	if strings.TrimSpace(reply[:loc[0]]) != "value mentions node $ inside" {
		t.Fatalf("body corrupted: %q", reply)
	}
}

func TestCommandClearsPrefixAndRemovesTrailingPrompt(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	go func() { defer b.Close(); buf := make([]byte, 6); b.Read(buf); fmt.Fprint(b, "result\nnode $ \n") }()
	c := &Client{conn: a, prompt: "node", preprompt: "[configure] "}
	reply, err := c.Command(context.Background(), "noconf")
	if err != nil || reply != "result" || c.Prompt() != "node $ " {
		t.Fatalf("%q %q %v", reply, c.Prompt(), err)
	}
}

func TestStreamArrivesBeforePromptAndPreservesPartialFailure(t *testing.T) {
	for _, complete := range []bool{true, false} {
		t.Run(fmt.Sprint(complete), func(t *testing.T) {
			a, b := net.Pipe()
			defer a.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			snapshots := make(chan string, 8)
			release := make(chan struct{})
			go func() {
				defer b.Close()
				command := make([]byte, 4)
				if _, err := io.ReadFull(b, command); err != nil {
					return
				}
				fmt.Fprint(b, "first\n  second")
				select {
				case <-release:
				case <-ctx.Done():
					return
				}
				if complete {
					fmt.Fprint(b, "\n[configure] no")
					fmt.Fprint(b, "de $ ")
				}
			}()
			c := &Client{conn: a, prompt: "node"}
			type result struct {
				output string
				err    error
			}
			done := make(chan result, 1)
			go func() {
				output, err := c.StreamCommand(ctx, "help", func(output string) { snapshots <- output })
				done <- result{output, err}
			}()
			select {
			case first := <-snapshots:
				if first != "first\n  second" {
					t.Fatal(first)
				}
			case <-ctx.Done():
				t.Fatal("no output before prompt")
			}
			select {
			case <-done:
				t.Fatal("command completed before prompt")
			default:
			}
			close(release)
			got := <-done
			if complete {
				if got.err != nil || got.output != "first\n  second\n" || c.Prompt() != "[configure] node $ " {
					t.Fatalf("%+v prompt=%q", got, c.Prompt())
				}
			} else if got.err == nil || got.output != "first\n  second" {
				t.Fatalf("lost partial output: %+v", got)
			}
		})
	}
}

func TestStreamCancellationUnblocksRead(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { buf := make([]byte, 4); io.ReadFull(b, buf); fmt.Fprint(b, "working\n") }()
	done := make(chan error, 1)
	go func() {
		_, err := (&Client{conn: a, prompt: "node"}).StreamCommand(ctx, "help", func(string) { cancel() })
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected cancellation")
		}
	case <-time.After(time.Second):
		t.Fatal("read not cancelled")
	}
}

func TestResponseWordsAreNotProtocolErrors(t *testing.T) {
	for _, reply := range []string{"AuthFailed is a message", "log says Bye!", "AuthFailed\nnode $ "} {
		t.Run(reply, func(t *testing.T) {
			a, b := net.Pipe()
			defer a.Close()
			go func() {
				defer b.Close()
				fmt.Fprint(b, reply)
				if !strings.HasSuffix(reply, "node $ ") {
					fmt.Fprint(b, "\nnode $ ")
				}
			}()
			c := &Client{conn: a, prompt: "node"}
			if _, err := c.receive(context.Background()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestProtocolAuthenticationAndCloseErrors(t *testing.T) {
	for _, tc := range []struct{ reply, prefix, want string }{
		{"AuthFailed\nBye!", "[password] ", "authentication failed"},
		{"Bye!", "", "session closed"},
		{"Connection is busy. Bye!", "", "server busy"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			a, b := net.Pipe()
			defer a.Close()
			go func() { defer b.Close(); fmt.Fprint(b, tc.reply) }()
			c := &Client{conn: a, prompt: "node", preprompt: tc.prefix}
			if _, err := c.receive(context.Background()); err == nil || err.Error() != tc.want {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestFragmentedProtocolWordsRemainResponseText(t *testing.T) {
	for _, word := range []string{"Bye!", "Connection is busy. Bye!", "AuthFailed"} {
		for _, suffix := range []string{" is an example message", "\nmore output"} {
			t.Run(word+suffix, func(t *testing.T) {
				a, b := net.Pipe()
				defer a.Close()
				defer b.Close()
				go func() {
					defer b.Close()
					command := make([]byte, 4)
					io.ReadFull(b, command)
					fmt.Fprint(b, word)
					fmt.Fprint(b, suffix+"\nnode $ ")
				}()
				// Include the authentication prefix to exercise both classifiers.
				c := &Client{conn: a, prompt: "node", preprompt: "[password] "}
				if word == "AuthFailed" && strings.HasPrefix(suffix, "\n") {
					c.preprompt = ""
				}
				var streamed string
				output, err := c.StreamCommand(context.Background(), "help", func(s string) { streamed = s })
				want := word + suffix + "\n"
				if err != nil || output != want || streamed != want {
					t.Fatalf("output=%q streamed=%q error=%v", output, streamed, err)
				}
			})
		}
	}
}

func TestResponseLimitAppliesBeforeFinalPrompt(t *testing.T) {
	// Exercise the same boundary with a small limit; this is not a throughput test.
	const limit = 1024
	a, b := net.Pipe()
	defer a.Close()
	go func() { defer b.Close(); fmt.Fprint(b, strings.Repeat("x", limit-4)); fmt.Fprint(b, "\nnode $ ") }()
	output, err := (&Client{conn: a, prompt: "node"}).receiveStreamLimit(context.Background(), nil, limit)
	if err == nil || err.Error() != "response exceeds 1 MiB" || len(output) > limit {
		t.Fatalf("size=%d error=%v", len(output), err)
	}
}
