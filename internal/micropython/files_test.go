package micropython

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestUploadsUseIndependentScratchPaths(t *testing.T) {
	data := []byte("updated application")
	reply := func(output string) string { return ">R\x00OK" + output + "\x04\x04" }
	response := reply("") + reply("") + reply("") +
		reply(fmt.Sprintf("%s%x\r\n", hashMarker, sha256.Sum256(data))) + reply("")
	port := newScriptedPort([]byte(response + response))
	client := newClient(port, time.Second)
	for range 2 {
		if err := client.WriteFileAtomic(context.Background(), "/main.py", data); err != nil {
			t.Fatal(err)
		}
	}
	writes := port.writes.String()
	paths := regexp.MustCompile(`_f=open\("([^"]+)",'wb'\)`).FindAllStringSubmatch(writes, -1)
	if len(paths) != 2 || paths[0][1] == paths[1][1] {
		t.Fatalf("uploads reused scratch files: %v", paths)
	}
	for _, match := range paths {
		if path.Dir(match[1]) != "/" || match[1] == "/main.py" {
			t.Fatalf("invalid scratch destination: %s", match[1])
		}
	}
	for _, name := range []string{"main.py.micros-update", "main.py.micros-backup"} {
		if strings.Contains(writes, name) {
			t.Fatalf("upload still touches a pre-existing recovery path: %s", name)
		}
	}
}

func TestScratchCollisionStopsBeforeUpload(t *testing.T) {
	port := newScriptedPort([]byte(">R\x00OK\x04OSError: upload scratch path already exists\r\n\x04"))
	err := newClient(port, time.Second).WriteFileAtomic(context.Background(), "/main.py", []byte("new data"))
	if err == nil || !strings.Contains(err.Error(), "scratch path already exists") {
		t.Fatalf("scratch collision was ignored: %v", err)
	}
	if strings.Contains(port.writes.String(), "_f.write") || strings.Contains(port.writes.String(), "os.rename") {
		t.Fatal("upload continued after a scratch collision")
	}
}
