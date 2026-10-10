package network

import (
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func TestProtectedStatusWithoutPassword(t *testing.T) {
	for _, hello := range []string{"hello:Tinydevboard:uid123:rel", "hello:other-board:uid456", "hello:broken:"} {
		t.Run(hello, func(t *testing.T) {
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
				command := make([]byte, len("hello"))
				if _, err := io.ReadFull(conn, command); err != nil {
					done <- err
					return
				}
				if string(command) != "hello" {
					done <- fmt.Errorf("unexpected command %q", command)
					return
				}
				fmt.Fprint(conn, hello+"\n[pass")
				fmt.Fprint(conn, "word] node $ ")
				// Discovery must close without issuing protected commands.
				n, err := conn.Read(command)
				if n != 0 || err != io.EOF {
					done <- fmt.Errorf("expected close after hello: bytes=%q error=%v", command[:n], err)
					return
				}
				done <- nil
			}()
			node, valid := inspect(context.Background(), Device{Address: listener.Addr().String(), Version: "old", Features: map[string]string{"webui": "ON"}}, "")
			if hello == "hello:broken:" {
				if valid || node.Online || node.Error != "not a micrOS hello response" {
					t.Fatalf("accepted invalid identity: %+v", node)
				}
			} else if !valid || !node.Online || node.Name == "" || node.UID == "" || node.Version != "n/a" || node.Features["auth"] != "ON" || len(node.Features) != 1 || node.Error != "" {
				t.Fatalf("unexpected protected node: %+v", node)
			}
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		})
	}
}
