package network

import (
	"context"
	"sync"
	"testing"
)

func TestSpecialEndpointsMergeByUIDAndStayHiddenWhenCached(t *testing.T) {
	const uid = "micr303862363166336236643138OS"
	lan := Device{Address: "192.168.1.10:9008", UID: uid, Name: "simulator", Online: true}
	store := &removalStore{nodes: []Device{
		lan,
		{Name: "simulator", Address: LocalhostAddress, UID: uid, Online: true},
	}}
	s := &Service{CIDR: "192.168.4.1/32", Store: store}
	cached, err := s.Cached()
	if err != nil || len(cached) != 0 {
		t.Fatalf("cached aliases = %+v, err = %v", cached, err)
	}
	for _, online := range []bool{true, false} {
		var mu sync.Mutex
		calls := map[string]int{}
		s.inspect = func(_ context.Context, d Device, _ string) (Device, bool) {
			mu.Lock()
			calls[d.Address]++
			mu.Unlock()
			d.Online, d.Cached = online, false
			if online {
				d.UID, d.Name = uid, "simulator"
			}
			return d, online
		}
		var observed []Device
		if err := s.Discover(context.Background(), func(d Device) { observed = append(observed, d) }); err != nil {
			t.Fatal(err)
		}
		if len(observed) != 3 || len(calls) != 3 {
			t.Fatalf("online=%v: observed %+v, calls %v", online, observed, calls)
		}
		for _, address := range []string{LocalhostAddress, APAddress, lan.Address} {
			if calls[address] != 1 {
				t.Fatalf("%s probed %d times", address, calls[address])
			}
		}
		if len(store.nodes) != 1 || store.nodes[0].Address != LocalhostAddress || store.nodes[0].UID != uid {
			t.Fatalf("cache must retain one canonical identity: %+v", store.nodes)
		}
		restarted := &Service{Store: store}
		if nodes, err := restarted.Cached(); err != nil || len(nodes) != 0 {
			t.Fatalf("restart resurrected simulator: %+v, %v", nodes, err)
		}
	}
}

func TestSpecialWebUIURLs(t *testing.T) {
	for _, tc := range []struct{ address, want string }{
		{LocalhostAddress, "http://localhost"},
		{APAddress, "http://192.168.4.1"},
	} {
		d := Device{Address: tc.address, Name: "simulator", Features: map[string]string{"webui": "ON"}}
		if got := d.WebUIURL(); got != tc.want {
			t.Fatalf("%s URL = %q, want %q", tc.address, got, tc.want)
		}
		d.Features["webui"] = "OFF"
		if got := d.WebUIURL(); got != "" {
			t.Fatalf("disabled WebUI URL = %q", got)
		}
	}
}
