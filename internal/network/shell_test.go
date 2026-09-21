package network

import (
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func TestShellIdentityAndPersistentMode(t *testing.T) {
	for _, uid := range []string{"id", "wrong"} {
		t.Run(uid, func(t *testing.T) {
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
				fmt.Fprint(conn, "[password] node $ ")
				commands := []string{"secret", "hello"}
				replies := []string{"AuthOk\nnode $ ", "hello:node:id:rel\nnode $ "}
				if uid == "id" {
					commands = append(commands, "conf", "webui", "noconf", "exit")
					replies = append(replies, "[configure] node $ ", "True\n[configure] node $ ", "node $ ", "")
				}
				for i, command := range commands {
					buf := make([]byte, len(command))
					if _, err := io.ReadFull(conn, buf); err != nil {
						done <- err
						return
					}
					if string(buf) != command {
						done <- fmt.Errorf("got %q want %q", buf, command)
						return
					}
					if _, err := fmt.Fprint(conn, replies[i]); err != nil {
						done <- err
						return
					}
				}
				done <- nil
			}()
			service := &Service{Password: "must-not-be-used"}
			session, err := service.OpenShell(context.Background(), Device{UID: uid, Address: listener.Addr().String()})
			if err != nil {
				t.Fatal(err)
			}
			if session.Prompt() != "[password] node $ " {
				t.Fatal(session.Prompt())
			}
			_, err = session.Command(context.Background(), "secret")
			if uid == "wrong" {
				if err == nil {
					session.Close()
					t.Fatal("accepted wrong identity")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				defer session.Close()
				if _, err := session.Command(context.Background(), "conf"); err != nil {
					t.Fatal(err)
				}
				if session.Prompt() != "[configure] node $ " {
					t.Fatal(session.Prompt())
				}
				if reply, err := session.Command(context.Background(), "webui"); err != nil || reply != "True" {
					t.Fatalf("reply %q: %v", reply, err)
				}
				if _, err := session.Command(context.Background(), "noconf"); err != nil {
					t.Fatal(err)
				}
				if session.Prompt() != "node $ " {
					t.Fatal(session.Prompt())
				}
				if _, err := session.Command(context.Background(), "exit"); err != ErrSessionClosed {
					t.Fatalf("exit: %v", err)
				}
			}
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		})
	}
}
