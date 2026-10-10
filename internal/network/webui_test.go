package network

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebUIURL(t *testing.T) {
	for _, tc := range []struct{ name, feature, want string }{
		{"Entrance", "ON", "http://Entrance.local"},
		{"Entrance", "OFF", ""},
		{"Entrance", "", ""},
		{"node-1", "ON", "http://node-1.local"},
		{"", "ON", ""},
		{"bad/name", "ON", ""},
		{"node.local@evil", "ON", ""},
		{"-node", "ON", ""},
	} {
		t.Run(tc.name+tc.feature, func(t *testing.T) {
			d := Device{Name: tc.name, Features: map[string]string{"webui": tc.feature}}
			if got := d.WebUIURL(); got != tc.want {
				t.Fatalf("WebUIURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestProtectedWebUIProbe(t *testing.T) {
	for _, status := range []int{200, 401, 403, 404, 500, 302} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/" || r.Header.Get("Authorization") != "" {
					t.Errorf("unexpected probe: %s %s", r.Method, r.URL)
				}
				w.Header().Set("Location", "/redirect")
				w.WriteHeader(status)
			}))
			defer server.Close()
			want := status == 200 || status == 401 || status == 403
			if got := webUIAvailable(context.Background(), server.URL); got != want {
				t.Fatalf("available=%v, want %v", got, want)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if webUIAvailable(ctx, server.URL) {
				t.Fatal("cancelled probe succeeded")
			}
		})
	}
	server := httptest.NewServer(http.NotFoundHandler())
	endpoint := server.URL
	server.Close()
	if webUIAvailable(context.Background(), endpoint) {
		t.Fatal("closed endpoint available")
	}
}

func TestDefaultWebUIURL(t *testing.T) {
	for _, tc := range []struct{ address, want string }{
		{"192.168.1.42:9008", "http://192.168.1.42"},
		{"localhost:9008", "http://localhost"},
		{"[::1]:9008", "http://[::1]"},
		{"invalid", ""},
	} {
		if got := defaultWebUIURL(tc.address); got != tc.want {
			t.Fatalf("%s: %q", tc.address, got)
		}
	}
	node := Device{Name: "node", Online: true, Features: map[string]string{"auth": "ON", "webui": "ON"}, ProbedWebUI: "http://192.168.1.42"}
	if got := node.WebUIURL(); got != node.ProbedWebUI {
		t.Fatal(got)
	}
	node.Features["webui"] = ""
	if got := node.WebUIURL(); got != "" {
		t.Fatal(got)
	}
}
